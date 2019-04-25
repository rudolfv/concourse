package lidar

import (
	"context"
	"os"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/atc/db"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
)

//go:generate counterfeiter . Notifications

type Notifications interface {
	Listen(channel string) (chan bool, error)
	Unlisten(channel string, notifier chan bool) error
}

//go:generate counterfeiter . Scanner

type Scanner interface {
	Scan(context.Context) error
}

//go:generate counterfeiter . Checker

type Checker interface {
	Check(context.Context, db.ResourceCheck) error
}

func NewRunner(
	logger lager.Logger,
	scanner Scanner,
	scanInterval time.Duration,
	checker Checker,
	checkInterval time.Duration,
	factory db.ResourceFactory,
	notifications Notifications,
) ifrit.Runner {
	return grouper.NewParallel(
		os.Interrupt,
		[]grouper.Member{
			{
				Name: "scanner",
				Runner: NewScanRunner(
					logger,
					scanInterval,
					scanner,
					factory,
					notifications,
				),
			},
			{
				Name: "checker",
				Runner: NewCheckRunner(
					logger,
					checkInterval,
					checker,
					factory,
					notifications,
				),
			},
		},
	)
}

func NewScanRunner(
	logger lager.Logger,
	scanInterval time.Duration,
	scanner Scanner,
	factory db.ResourceFactory,
	notifications Notifications,
) ifrit.Runner {
	return &scanRunner{
		logger,
		scanInterval,
		scanner,
		factory,
		notifications,
		&sync.WaitGroup{},
	}
}

type scanRunner struct {
	logger        lager.Logger
	interval      time.Duration
	scanner       Scanner
	factory       db.ResourceFactory
	notifications Notifications
	waitGroup     *sync.WaitGroup
}

func (r *scanRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	r.logger.Info("start")
	defer r.logger.Info("done")

	close(ready)

	ticker := time.NewTicker(r.interval)
	ctx, cancel := context.WithCancel(context.Background())

	r.run(ctx)

	for {
		select {
		case <-ticker.C:
			r.run(ctx)
		case <-signals:
			cancel()
			r.waitGroup.Wait()
			return nil
		}
	}
}

func (r *scanRunner) run(ctx context.Context) {
	r.waitGroup.Add(1)
	defer r.waitGroup.Done()

	if err := r.scanner.Scan(ctx); err != nil {
		r.logger.Error("failed-to-scan", err)
	}
}

func NewCheckRunner(
	logger lager.Logger,
	checkInterval time.Duration,
	checker Checker,
	factory db.ResourceFactory,
	notifications Notifications,
) ifrit.Runner {
	return &checkRunner{
		logger,
		checkInterval,
		checker,
		factory,
		notifications,
		&sync.WaitGroup{},
	}
}

type checkRunner struct {
	logger        lager.Logger
	interval      time.Duration
	checker       Checker
	factory       db.ResourceFactory
	notifications Notifications
	waitGroup     *sync.WaitGroup
}

func (r *checkRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	r.logger.Info("start")
	defer r.logger.Info("done")

	close(ready)

	channel := "check_resources"
	notifier, err := r.notifications.Listen(channel)
	if err != nil {
		return err
	}

	defer r.notifications.Unlisten(channel, notifier)

	ticker := time.NewTicker(r.interval)
	ctx, cancel := context.WithCancel(context.Background())

	if err = r.checkAll(ctx); err != nil {
		r.logger.Error("failed-to-check-all", err)
	}

	for {
		select {
		case <-ticker.C:
			if err = r.checkAll(ctx); err != nil {
				r.logger.Error("failed-to-check-all", err)
			}
		case <-notifier:
			if err = r.checkAll(ctx); err != nil {
				r.logger.Error("failed-to-check-all", err)
			}
		case <-signals:
			cancel()
			r.waitGroup.Wait()
			return nil
		}
	}
}

func (r *checkRunner) checkAll(ctx context.Context) error {

	resourceChecks, err := r.factory.ResourceChecks()
	if err != nil {
		return err
	}

	for _, resourceCheck := range resourceChecks {
		go r.check(ctx, resourceCheck)
	}

	return nil
}

func (r *checkRunner) check(ctx context.Context, resourceCheck db.ResourceCheck) {
	r.waitGroup.Add(1)
	defer r.waitGroup.Done()

	if err := r.checker.Check(ctx, resourceCheck); err != nil {
		r.logger.Error("failed-to-check", err)
	}
}
