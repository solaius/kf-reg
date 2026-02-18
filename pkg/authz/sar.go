package authz

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SARAuthorizer checks authorization using Kubernetes SubjectAccessReview.
type SARAuthorizer struct {
	client kubernetes.Interface
}

// NewSARAuthorizer creates a new SARAuthorizer backed by the given Kubernetes client.
func NewSARAuthorizer(client kubernetes.Interface) *SARAuthorizer {
	return &SARAuthorizer{client: client}
}

// Authorize performs a SubjectAccessReview check against the Kubernetes API server.
func (s *SARAuthorizer) Authorize(ctx context.Context, req AuthzRequest) (bool, error) {
	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   req.User,
			Groups: req.Groups,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group:    APIGroup,
				Resource: req.Resource,
				Verb:     req.Verb,
			},
		},
	}

	if req.Namespace != "" {
		sar.Spec.ResourceAttributes.Namespace = req.Namespace
	}

	review, err := s.client.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("SAR check failed: %w", err)
	}

	return review.Status.Allowed, nil
}
