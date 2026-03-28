package billing

import (
	"sync"
	"sync/atomic"

	billing_repositories "databasus-backend/internal/features/billing/repositories"
	"databasus-backend/internal/features/databases"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

var (
	billingService = &BillingService{
		&billing_repositories.SubscriptionRepository{},
		&billing_repositories.SubscriptionEventRepository{},
		&billing_repositories.InvoiceRepository{},
		nil, // billing provider will be set later to avoid circular dependency
		workspaces_services.GetWorkspaceService(),
		*databases.GetDatabaseService(),
		atomic.Bool{},
	}
	billingController = &BillingController{billingService}
)

func GetBillingService() *BillingService {
	return billingService
}

func GetBillingController() *BillingController {
	return billingController
}

var SetupDependencies = sync.OnceFunc(func() {
	databases.GetDatabaseService().AddDbCreationListener(billingService)
})
