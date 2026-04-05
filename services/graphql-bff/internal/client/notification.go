package client

import (
	"context"
	"fmt"

	notificationpb "graphql-bff/api/gen/notification"

	"google.golang.org/grpc"
)

type NotificationClient struct {
	stub notificationpb.NotificationServiceClient
}

func NewNotificationClient(conn *grpc.ClientConn) *NotificationClient {
	return &NotificationClient{stub: notificationpb.NewNotificationServiceClient(conn)}
}

func (c *NotificationClient) Send(ctx context.Context, req *notificationpb.SendNotificationRequest) (string, notificationpb.NotificationStatus, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.SendNotification(ctx, req)
	if err != nil {
		return "", notificationpb.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED, fmt.Errorf("notification.SendNotification: %w", err)
	}
	return resp.NotificationId, resp.Status, nil
}

func (c *NotificationClient) GetDeliveryStatus(ctx context.Context, notificationID, tenantID string) (*notificationpb.Notification, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetDeliveryStatus(ctx, &notificationpb.GetDeliveryStatusRequest{
		NotificationId: notificationID,
		TenantId:       tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("notification.GetDeliveryStatus: %w", err)
	}
	return resp.Notification, nil
}

func (c *NotificationClient) UpdatePreferences(ctx context.Context, req *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.NotificationPreference, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.UpdateNotificationPreferences(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("notification.UpdateNotificationPreferences: %w", err)
	}
	return resp.Preference, nil
}
