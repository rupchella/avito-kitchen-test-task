package restaurant

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const ordersPerPoll = 100

const (
	statusCreated   = "created"
	statusAccepted  = "accepted"
	statusRejected  = "rejected"
	statusPreparing = "preparing"
	statusReady     = "ready"
)

type Worker struct {
	client          *Client
	logger          *slog.Logger
	menu            []MenuItemInput
	interval        time.Duration
	maxOrderKopecks int64
	menuSynced      bool
}

func NewWorker(
	client *Client,
	logger *slog.Logger,
	menu []MenuItemInput,
	interval time.Duration,
	maxOrderKopecks int64,
) *Worker {
	return &Worker{
		client:          client,
		logger:          logger,
		menu:            menu,
		interval:        interval,
		maxOrderKopecks: maxOrderKopecks,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if !w.menuSynced {
		w.syncMenu(ctx)
	}

	w.advanceOrders(ctx, statusPreparing, statusReady)
	w.advanceOrders(ctx, statusAccepted, statusPreparing)
	w.processNewOrders(ctx)
}

func (w *Worker) syncMenu(ctx context.Context) {
	items, err := w.client.SyncMenu(ctx, w.menu)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			w.logger.Warn("failed to sync menu", "error", err)
		}

		return
	}

	w.menuSynced = true

	w.logger.Info("menu synchronized", "items", len(items))
}

func (w *Worker) processNewOrders(ctx context.Context) {
	orders, err := w.client.ListOrders(ctx, statusCreated, ordersPerPoll)
	if err != nil {
		w.logOrderError("failed to fetch new orders", err)

		return
	}

	for _, order := range orders {
		target := statusAccepted

		if order.TotalPriceKopecks > w.maxOrderKopecks {
			target = statusRejected
		}

		w.updateStatus(ctx, order, target)
	}
}

func (w *Worker) advanceOrders(
	ctx context.Context,
	from string,
	to string,
) {
	orders, err := w.client.ListOrders(ctx, from, ordersPerPoll)
	if err != nil {
		w.logOrderError("failed to fetch orders", err)

		return
	}

	for _, order := range orders {
		w.updateStatus(ctx, order, to)
	}
}

func (w *Worker) updateStatus(
	ctx context.Context,
	order Order,
	target string,
) {
	if _, err := w.client.UpdateOrderStatus(ctx, order.ID, target); err != nil {
		w.logOrderError("failed to update order status", err)

		return
	}

	w.logger.Info(
		"order status updated",
		"order_id", order.ID,
		"status", target,
	)
}

func (w *Worker) logOrderError(message string, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}

	w.logger.Warn(message, "error", err)
}
