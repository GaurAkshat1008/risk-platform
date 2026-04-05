// Package client holds gRPC client connections to all downstream services.
package client

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Registry bundles every downstream gRPC client used by the BFF resolvers.
type Registry struct {
	Decision     *DecisionClient
	CaseMgmt     *CaseManagementClient
	Ingestion    *IngestionClient
	Explanation  *ExplanationClient
	Workflow     *WorkflowClient
	Audit        *AuditClient
	Notification *NotificationClient
	OpsQuery     *OpsQueryClient
	Rules        *RulesClient
	Tenant       *TenantClient
	Identity     *IdentityClient

	conns []*grpc.ClientConn
}

type Addrs struct {
	Decision     string
	CaseMgmt     string
	Ingestion    string
	Explanation  string
	Workflow     string
	Audit        string
	Notification string
	OpsQuery     string
	Rules        string
	Tenant       string
	Identity     string
}

func NewRegistry(a Addrs) (*Registry, error) {
	dial := func(name, addr string) (*grpc.ClientConn, error) {
		if addr == "" {
			return nil, fmt.Errorf("%s: empty address", name)
		}
		return grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
	}

	type entry struct {
		name string
		addr string
	}

	addrs := []entry{
		{"decision", a.Decision},
		{"case-management", a.CaseMgmt},
		{"ingestion", a.Ingestion},
		{"explanation", a.Explanation},
		{"workflow", a.Workflow},
		{"audit-trail", a.Audit},
		{"notification", a.Notification},
		{"ops-query", a.OpsQuery},
		{"rules-engine", a.Rules},
		{"tenant-config", a.Tenant},
		{"identity-access", a.Identity},
	}

	conns := make([]*grpc.ClientConn, 0, len(addrs))
	for _, e := range addrs {
		c, err := dial(e.name, e.addr)
		if err != nil {
			for _, oc := range conns {
				_ = oc.Close()
			}
			return nil, err
		}
		conns = append(conns, c)
	}

	return &Registry{
		Decision:     NewDecisionClient(conns[0]),
		CaseMgmt:     NewCaseManagementClient(conns[1]),
		Ingestion:    NewIngestionClient(conns[2]),
		Explanation:  NewExplanationClient(conns[3]),
		Workflow:     NewWorkflowClient(conns[4]),
		Audit:        NewAuditClient(conns[5]),
		Notification: NewNotificationClient(conns[6]),
		OpsQuery:     NewOpsQueryClient(conns[7]),
		Rules:        NewRulesClient(conns[8]),
		Tenant:       NewTenantClient(conns[9]),
		Identity:     NewIdentityClient(conns[10]),
		conns:        conns,
	}, nil
}

// Close shuts down all gRPC connections.
func (r *Registry) Close() {
	for _, c := range r.conns {
		_ = c.Close()
	}
}
