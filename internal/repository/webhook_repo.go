package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type WebhookRepository struct{ store *storage.JSONStore }

func NewWebhookRepository(store *storage.JSONStore) *WebhookRepository {
	return &WebhookRepository{store: store}
}

func (r *WebhookRepository) allEndpoints() ([]model.WebhookEndpoint, error) {
	var items []model.WebhookEndpoint
	err := r.store.Load("webhooks", &items)
	return items, err
}
func (r *WebhookRepository) saveEndpoints(items []model.WebhookEndpoint) error {
	return r.store.Save("webhooks", items)
}

func (r *WebhookRepository) CreateEndpoint(endpoint model.WebhookEndpoint) error {
	items, err := r.allEndpoints()
	if err != nil {
		return err
	}
	items = append(items, endpoint)
	return r.saveEndpoints(items)
}

func (r *WebhookRepository) UpdateEndpoint(endpoint model.WebhookEndpoint) error {
	items, err := r.allEndpoints()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == endpoint.ID {
			items[i] = endpoint
			return r.saveEndpoints(items)
		}
	}
	items = append(items, endpoint)
	return r.saveEndpoints(items)
}

func (r *WebhookRepository) GetEndpointByID(id string) (model.WebhookEndpoint, bool, error) {
	items, err := r.allEndpoints()
	if err != nil {
		return model.WebhookEndpoint{}, false, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, true, nil
		}
	}
	return model.WebhookEndpoint{}, false, nil
}

func (r *WebhookRepository) ListEndpoints() ([]model.WebhookEndpoint, error) { return r.allEndpoints() }

func (r *WebhookRepository) ListEnabledEndpoints() ([]model.WebhookEndpoint, error) {
	items, err := r.allEndpoints()
	if err != nil {
		return nil, err
	}
	out := []model.WebhookEndpoint{}
	for _, item := range items {
		if item.Status == model.WebhookStatusEnabled {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *WebhookRepository) DeleteEndpoint(id string) error {
	items, err := r.allEndpoints()
	if err != nil {
		return err
	}
	out := items[:0]
	for _, item := range items {
		if item.ID != id {
			out = append(out, item)
		}
	}
	return r.saveEndpoints(out)
}

func (r *WebhookRepository) allDeliveries() ([]model.WebhookDelivery, error) {
	var items []model.WebhookDelivery
	err := r.store.Load("webhook_deliveries", &items)
	return items, err
}
func (r *WebhookRepository) saveDeliveries(items []model.WebhookDelivery) error {
	return r.store.Save("webhook_deliveries", items)
}

func (r *WebhookRepository) CreateDelivery(delivery model.WebhookDelivery) error {
	items, err := r.allDeliveries()
	if err != nil {
		return err
	}
	items = append(items, delivery)
	return r.saveDeliveries(items)
}

func (r *WebhookRepository) UpdateDelivery(delivery model.WebhookDelivery) error {
	items, err := r.allDeliveries()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == delivery.ID {
			items[i] = delivery
			return r.saveDeliveries(items)
		}
	}
	items = append(items, delivery)
	return r.saveDeliveries(items)
}

func (r *WebhookRepository) ListDeliveries(webhookID string) ([]model.WebhookDelivery, error) {
	items, err := r.allDeliveries()
	if err != nil {
		return nil, err
	}
	out := []model.WebhookDelivery{}
	for _, item := range items {
		if webhookID == "" || item.WebhookID == webhookID {
			out = append(out, item)
		}
	}
	return out, nil
}
