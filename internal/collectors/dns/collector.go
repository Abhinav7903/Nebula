package dns

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct{}

func (c *Collector) Name() string             { return "dns" }
func (c *Collector) SupportedTypes() []string  { return []string{"domain", "subdomain", "url"} }
func (c *Collector) RequiresKey() bool         { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	resolver := net.DefaultResolver
	var results []collectors.Result

	recordTypes := []string{"A", "AAAA", "MX", "NS", "TXT", "CNAME", "SOA"}
	for _, rt := range recordTypes {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		records, err := resolveType(ctx, resolver, query, rt)
		if err != nil {
			continue
		}
		for _, r := range records {
			results = append(results, collectors.Result{
				ID:          uuid.NewString(),
				Collector:   "dns",
				Type:        "dns_record",
				Title:       rt + " record for " + query,
				Description: r,
				Data: map[string]any{
					"type":   rt,
					"value":  r,
					"domain": query,
				},
				Tags:       []string{"dns", rt},
				Confidence: 1.0,
				Source:     "dns_resolver",
				FoundAt:    time.Now(),
			})
		}
	}
	return results, nil
}

func resolveType(ctx context.Context, r *net.Resolver, domain, rt string) ([]string, error) {
	switch rt {
	case "A":
		ips, err := r.LookupHost(ctx, domain)
		if err != nil {
			return nil, err
		}
		return ips, nil
	case "AAAA":
		ips, err := r.LookupHost(ctx, domain)
		if err != nil {
			return nil, err
		}
		return ips, nil
	case "MX":
		mxes, err := r.LookupMX(ctx, domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(mxes))
		for i, mx := range mxes {
			out[i] = mx.Host
		}
		return out, nil
	case "NS":
		nss, err := r.LookupNS(ctx, domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(nss))
		for i, ns := range nss {
			out[i] = ns.Host
		}
		return out, nil
	case "TXT":
		txts, err := r.LookupTXT(ctx, domain)
		if err != nil {
			return nil, err
		}
		return txts, nil
	case "CNAME":
		cname, err := r.LookupCNAME(ctx, domain)
		if err != nil {
			return nil, err
		}
		return []string{cname}, nil
	case "SOA":
		return nil, nil
	default:
		return nil, nil
	}
}
