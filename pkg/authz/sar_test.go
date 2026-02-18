package authz

import (
	"context"
	"testing"

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestSARAuthorizer(t *testing.T) {
	tests := []struct {
		name          string
		sarAllowed    bool
		req           AuthzRequest
		wantAllowed   bool
		wantErr       bool
	}{
		{
			name:       "allowed - namespace scoped",
			sarAllowed: true,
			req: AuthzRequest{
				User:      "alice",
				Groups:    []string{"team-a"},
				Resource:  ResourceCatalogSources,
				Verb:      VerbCreate,
				Namespace: "team-a",
			},
			wantAllowed: true,
		},
		{
			name:       "denied - namespace scoped",
			sarAllowed: false,
			req: AuthzRequest{
				User:      "bob",
				Groups:    []string{"team-b"},
				Resource:  ResourceCatalogSources,
				Verb:      VerbDelete,
				Namespace: "team-a",
			},
			wantAllowed: false,
		},
		{
			name:       "allowed - cluster scoped",
			sarAllowed: true,
			req: AuthzRequest{
				User:     "admin",
				Groups:   []string{"platform-ops"},
				Resource: ResourcePlugins,
				Verb:     VerbList,
			},
			wantAllowed: true,
		},
		{
			name:       "denied - cluster scoped",
			sarAllowed: false,
			req: AuthzRequest{
				User:     "viewer",
				Groups:   nil,
				Resource: ResourceActions,
				Verb:     VerbExecute,
			},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset()
			client.Fake.PrependReactor("create", "subjectaccessreviews",
				func(action k8stesting.Action) (bool, runtime.Object, error) {
					sar := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview)

					// Verify the SAR was constructed correctly.
					if sar.Spec.User != tt.req.User {
						t.Errorf("SAR User = %q, want %q", sar.Spec.User, tt.req.User)
					}
					if sar.Spec.ResourceAttributes.Group != APIGroup {
						t.Errorf("SAR Group = %q, want %q", sar.Spec.ResourceAttributes.Group, APIGroup)
					}
					if sar.Spec.ResourceAttributes.Resource != tt.req.Resource {
						t.Errorf("SAR Resource = %q, want %q", sar.Spec.ResourceAttributes.Resource, tt.req.Resource)
					}
					if sar.Spec.ResourceAttributes.Verb != tt.req.Verb {
						t.Errorf("SAR Verb = %q, want %q", sar.Spec.ResourceAttributes.Verb, tt.req.Verb)
					}
					if sar.Spec.ResourceAttributes.Namespace != tt.req.Namespace {
						t.Errorf("SAR Namespace = %q, want %q", sar.Spec.ResourceAttributes.Namespace, tt.req.Namespace)
					}

					sar.Status.Allowed = tt.sarAllowed
					return true, sar, nil
				},
			)

			authorizer := NewSARAuthorizer(client)
			allowed, err := authorizer.Authorize(context.Background(), tt.req)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if allowed != tt.wantAllowed {
				t.Errorf("allowed = %v, want %v", allowed, tt.wantAllowed)
			}
		})
	}
}
