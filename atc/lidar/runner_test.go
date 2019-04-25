package lidar_test

import (
	"errors"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/dbfakes"
	"github.com/concourse/concourse/atc/lidar"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Runner", func() {
	var (
		err     error
		runner  ifrit.Runner
		process ifrit.Process

		logger              *lagertest.TestLogger
		interval            time.Duration
		fakeScanner         *lidarfakes.FakeScanner
		fakeChecker         *lidarfakes.FakeChecker
		fakeNotifications   *lidarfakes.FakeNotifications
		fakeResourceFactory *dbfakes.FakeResourceFactory

		fakeClock *fakeclock.FakeClock
		runAt     time.Time
		runTimes  chan time.Time
	)
	var ()

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		interval = time.Minute
		fakeScanner = new(lidarfakes.FakeScanner)
		fakeChecker = new(lidarfakes.FakeChecker)
		fakeNotifications = new(lidarfakes.FakeNotifications)
		fakeResourceFactory = new(dbfakes.FakeResourceFactory)

		fakeClock = fakeclock.NewFakeClock(runAt)
		runAt = time.Unix(111, 111).UTC()
		runTimes = make(chan time.Time, 100)
	})

	JustBeforeEach(func() {
		process = ifrit.Invoke(runner)
	})

	AfterEach(func() {
		process.Signal(os.Interrupt)
		<-process.Wait()
	})

	Describe("ScanRunner", func() {

		BeforeEach(func() {
			runner = lidar.NewScanRunner(logger, interval, fakeScanner, fakeResourceFactory, fakeNotifications)
		})

		It("runs immediately", func() {
			Expect(fakeScanner.ScanCallCount()).To(Equal(1))
		})

		Context("when the interval elapses", func() {
			It("runs", func() {
			})
		})

		Context("when it receives shutdown signal", func() {
			BeforeEach(func() {
				go func() {
					process.Signal(os.Interrupt)
				}()
			})

			It("waits for things to finish", func() {
			})
		})
	})

	Describe("CheckRunner", func() {
		var fakeResourceCheck *dbfakes.FakeResourceCheck

		BeforeEach(func() {
			fakeResourceCheck = new(dbfakes.FakeResourceCheck)
			fakeResourceFactory.GetAllResourceChecksReturns([]db.ResourceCheck{fakeResourceCheck})

			runner = lidar.NewCheckRunner(logger, interval, fakeChecker, fakeResourceFactory, fakeNotifications)
		})

		Context("when listening for notifications fails", func() {
			BeforeEach(func() {
				fakeNotifications.ListenReturns(nil, errors.New("nope"))
			})

			It("errors", func() {
				Expect(<-process.Wait()).To(HaveOccured())
			})
		})

		Context("when listening for notifications succeeds", func() {
			var notify chan bool

			BeforeEach(func() {
				notify = make(chan bool)
				fakeNotifications.ListenReturns(notify, nil)

				fakeChecker.CheckStub = func() error {
					runTimes <- runAt
					return nil
				}
			})

			AfterEach(func() {
				<-process.Wait()
				Expect(fakeNotifications.UnlistenCallCount()).To(Equal(1))
			})

			BeforeEach(func() {
				notify = make(chan bool)
				fakeNotifications.ListenReturns(notify, nil)

				fakeChecker.CheckStub = func() error {
					runTimes <- runAt
					return nil
				}
			})

			It("runs once before the interval elapses", func() {
				Expect(<-runTimes).To(Equal(runAt))
			})

			Context("when the interval elapses", func() {
				It("runs", func() {
				})
			})

			Context("when it receives shutdown signal", func() {
				It("waits for things to finish", func() {
				})
			})
		})
	})
})
