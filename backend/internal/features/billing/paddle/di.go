package billing_paddle

import (
	"sync"

	"github.com/PaddleHQ/paddle-go-sdk"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/billing"
	billing_webhooks "databasus-backend/internal/features/billing/webhooks"
)

var (
	paddleBillingService    *PaddleBillingService
	paddleBillingController *PaddleBillingController
)

var initPaddle = sync.OnceFunc(func() {
	if config.GetEnv().IsPaddleSandbox {
		paddleClient, err := paddle.NewSandbox(config.GetEnv().PaddleApiKey)
		if err != nil {
			return
		}

		paddleBillingService = &PaddleBillingService{
			paddleClient,
			paddle.NewWebhookVerifier(config.GetEnv().PaddleWebhookSecret),
			config.GetEnv().PaddlePriceID,
			billing_webhooks.WebhookRepository{},
			billing.GetBillingService(),
		}
	} else {
		paddleClient, err := paddle.New(config.GetEnv().PaddleApiKey)
		if err != nil {
			return
		}

		paddleBillingService = &PaddleBillingService{
			paddleClient,
			paddle.NewWebhookVerifier(config.GetEnv().PaddleWebhookSecret),
			config.GetEnv().PaddlePriceID,
			billing_webhooks.WebhookRepository{},
			billing.GetBillingService(),
		}
	}

	paddleBillingController = &PaddleBillingController{paddleBillingService}
})

func GetPaddleBillingService() *PaddleBillingService {
	if !config.GetEnv().IsCloud {
		return nil
	}

	initPaddle()
	return paddleBillingService
}

func GetPaddleBillingController() *PaddleBillingController {
	if !config.GetEnv().IsCloud {
		return nil
	}

	// Ensure service + controller are initialized
	GetPaddleBillingService()

	return paddleBillingController
}

func SetupDependencies() {
	billing.GetBillingService().SetBillingProvider(GetPaddleBillingService())
}
