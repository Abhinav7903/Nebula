package geoip

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/oschwald/geoip2-golang"
	"github.com/yourusername/nebula/internal/collectors"
)

type Collector struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

func New(cityDBPath, asnDBPath string) (*Collector, error) {
	c := &Collector{}
	if cityDBPath != "" {
		db, err := geoip2.Open(cityDBPath)
		if err != nil {
			return nil, fmt.Errorf("open city DB: %w", err)
		}
		c.cityDB = db
	}
	if asnDBPath != "" {
		db, err := geoip2.Open(asnDBPath)
		if err != nil {
			return nil, fmt.Errorf("open ASN DB: %w", err)
		}
		c.asnDB = db
	}
	return c, nil
}

func (c *Collector) Name() string            { return "geoip" }
func (c *Collector) SupportedTypes() []string { return []string{"ipv4", "ipv6", "cidr"} }
func (c *Collector) RequiresKey() bool        { return false }

func (c *Collector) Execute(ctx context.Context, query string, qtype string) ([]collectors.Result, error) {
	var results []collectors.Result

	switch qtype {
	case "cidr":
		_, ipnet, err := net.ParseCIDR(query)
		if err != nil {
			return nil, err
		}
		results = append(results, collectors.Result{
			ID:          uuid.NewString(),
			Collector:   "geoip",
			Type:        "cidr_info",
			Title:       fmt.Sprintf("CIDR range: %s", ipnet.String()),
			Description: fmt.Sprintf("Network: %s, Mask: %s", ipnet.IP, net.IP(ipnet.Mask)),
			Data: map[string]any{
				"cidr":  ipnet.String(),
				"ip":    ipnet.IP.String(),
				"mask":  net.IP(ipnet.Mask).String(),
				"ones":  ones(ipnet.Mask),
				"bits":  bits(ipnet.Mask),
			},
			Tags:       []string{"geoip", "cidr", "network"},
			Confidence: 1.0,
			Source:     "maxmind",
			FoundAt:    time.Now(),
		})
		query = ipnet.IP.String()

	case "ipv4", "ipv6":
		parsed := net.ParseIP(query)
		if parsed == nil {
			return nil, fmt.Errorf("invalid IP: %s", query)
		}
		query = parsed.String()

	default:
		return nil, fmt.Errorf("unsupported type: %s", qtype)
	}

	ip := net.ParseIP(query)
	if ip == nil {
		return results, nil
	}

	if c.cityDB != nil {
		city, err := c.cityDB.City(ip)
		if err == nil {
			data := map[string]any{
				"ip":            query,
				"country":       city.Country.IsoCode,
				"country_name":  city.Country.Names["en"],
				"city":          city.City.Names["en"],
				"postal_code":   city.Postal.Code,
				"latitude":      city.Location.Latitude,
				"longitude":     city.Location.Longitude,
				"timezone":      city.Location.TimeZone,
				"continent":     city.Continent.Code,
			}
			cityName := city.City.Names["en"]
				countryName := city.Country.Names["en"]
				subName := ""
				subISO := ""
				if len(city.Subdivisions) > 0 {
					subName = city.Subdivisions[0].Names["en"]
					subISO = city.Subdivisions[0].IsoCode
					data["subdivision"] = subName
					data["subdivision_iso"] = subISO
				}
				desc := cityName
				if subName != "" {
					desc += ", " + subName
				}
				if countryName != "" {
					desc += ", " + countryName
				}

			results = append(results, collectors.Result{
				ID:          uuid.NewString(),
				Collector:   "geoip",
				Type:        "geoip_city",
				Title:       fmt.Sprintf("GeoIP city lookup for %s", query),
				Description: desc,
				Data:        data,
				Tags:        []string{"geoip", "city", "geolocation"},
				Confidence:  0.95,
				Source:      "maxmind_geolite2",
				FoundAt:     time.Now(),
			})
		}
	}

	if c.asnDB != nil {
		asn, err := c.asnDB.ASN(ip)
		if err == nil {
			results = append(results, collectors.Result{
				ID:          uuid.NewString(),
				Collector:   "geoip",
				Type:        "geoip_asn",
				Title:       fmt.Sprintf("ASN for %s: AS%d", query, asn.AutonomousSystemNumber),
				Description: fmt.Sprintf("%s (AS%d)", asn.AutonomousSystemOrganization, asn.AutonomousSystemNumber),
				Data: map[string]any{
					"ip":              query,
					"asn":             asn.AutonomousSystemNumber,
					"organization":    asn.AutonomousSystemOrganization,
				},
				Tags:       []string{"geoip", "asn", "network"},
				Confidence: 0.95,
				Source:     "maxmind_geolite2",
				FoundAt:    time.Now(),
			})
		}
	}

	return results, nil
}

func (c *Collector) Close() {
	if c.cityDB != nil {
		c.cityDB.Close()
	}
	if c.asnDB != nil {
		c.asnDB.Close()
	}
}

func ones(mask net.IPMask) int {
	ones, _ := mask.Size()
	return ones
}

func bits(mask net.IPMask) int {
	_, bits := mask.Size()
	return bits
}
