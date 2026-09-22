package notification

import (
	"context"
	"log/slog"
)

type Notifier interface {
	Send(ctx context.Context, recipientID int64, title, message string) error
}

type LogNotifier struct {
	logger *slog.Logger
}

func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

func (n *LogNotifier) Send(ctx context.Context, recipientID int64, title, message string) error {
	n.logger.Info("notification sent",
		slog.Int64("recipient_id", recipientID),
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}
