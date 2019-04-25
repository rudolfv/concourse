package lidar

import (
	"context"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
)

func NewChecker(
	logger lager.Logger,
	factory db.ResourceFactory,
) *checker {
	return &checker{
		logger,
		factory,
	}
}

type checker struct {
	logger  lager.Logger
	factory db.ResourceFactory
}

func (c *checker) Run(ctx context.Context) error {

	resourceChecks, err := c.factory.ResourceChecks()
	if err != nil {
		c.logger.Error("failed-to-fetch-resource-checks", err)
		return err
	}

	for _, resourceCheck := range resourceChecks {
		go c.tryCheck(ctx, resourceCheck)
	}

	return nil
}

func (c *checker) tryCheck(ctx context.Context, resourceCheck db.ResourceCheck) error {

	resource, err := resourceCheck.Resource()
	if err != nil {
		c.logger.Error("failed-to-fetch-resource", err)
		return err
	}

	parent, err := resource.ParentResourceType()
	if err != nil {
		c.logger.Error("failed-to-fetch-parent-type", err)
		return err
	}

	if parent.Version() == nil {
		if err = r.factory.CreateResourceCheck(parent.ID(), db.CheckTypeResourceType); err != nil {
			c.logger.Error("failed-to-request-parent-check", err)
			return err
		}

		if err = resourceCheck.Error("parent resource has no version"); err != nil {
			c.logger.Error("failed-to-save-resource-check-error", err)
			return err
		}
	}

	return nil
}

func (c *checker) check(ctx context.Context, resourceID int, fromVersion atc.Version) error {
	return nil
}
