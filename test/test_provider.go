package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"time"

	immosquare "github.com/immosquare/caddy-dns-immosquare"
	"github.com/libdns/libdns"
)

func main() {
	// Get token from environment variable
	apiToken := os.Getenv("API_TOKEN")
	endPoint := os.Getenv("ENDPOINT")
	if apiToken == "" {
		log.Fatal("Error: API_TOKEN environment variable not defined")
	}
	if endPoint == "" {
		log.Fatal("Error: ENDPOINT environment variable not defined")
	}

	// Provider configuration
	provider := &immosquare.Provider{
		APIToken: apiToken,
		Endpoint: endPoint,
	}

	ctx := context.Background()
	zone := "example.com"

	fmt.Println("=== Testing immosquare DNS provider ===")

	// Test 1: Get existing records
	fmt.Println("\n1. Retrieving existing records...")
	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		log.Printf("GetRecords error: %v", err)
	} else {
		fmt.Printf("✅ %d records found\n", len(records))
		for i, record := range records {
			rr := record.RR()
			fmt.Printf("  %d. %s %s %s (TTL: %s)\n", i+1, rr.Name, rr.Type, rr.Data, rr.TTL)
		}
	}

	// Test 2: Add a TXT record for ACME challenge
	fmt.Println("\n2. Adding a TXT record for ACME challenge...")
	newTXTRecord := libdns.TXT{
		Name: "_acme-challenge",
		Text: "test-challenge-token-12345",
		TTL:  300 * time.Second,
	}

	addedRecords, err := provider.AppendRecords(ctx, zone, []libdns.Record{newTXTRecord})
	if err != nil {
		log.Printf("AppendRecords error: %v", err)
	} else {
		fmt.Printf("✅ %d records added\n", len(addedRecords))
		for i, record := range addedRecords {
			rr := record.RR()
			fmt.Printf("  %d. %s %s %s (TTL: %s)\n",
				i+1, rr.Name, rr.Type, rr.Data, rr.TTL)
		}
	}

	// Test 3: Use SetRecords to define all records
	fmt.Println("\n3. Testing SetRecords (replace the given RRsets only)...")

	// Create valid IP addresses
	ip1, _ := netip.ParseAddr("192.99.250.180")
	ip2, _ := netip.ParseAddr("192.99.250.181")
	ip3, _ := netip.ParseAddr("192.99.250.182")

	setRecords := []libdns.Record{
		libdns.Address{
			Name: "www",
			IP:   ip1,
			TTL:  600 * time.Second,
		},
		libdns.Address{
			Name: "mail",
			IP:   ip2,
			TTL:  900 * time.Second,
		},
		libdns.Address{
			Name: "api",
			IP:   ip3,
			TTL:  1200 * time.Second,
		},
	}

	updatedRecords, err := provider.SetRecords(ctx, zone, setRecords)
	if err != nil {
		log.Printf("SetRecords error: %v", err)
	} else {
		fmt.Printf("✅ %d records defined\n", len(updatedRecords))
		for i, record := range updatedRecords {
			rr := record.RR()
			fmt.Printf("  %d. %s %s %s (TTL: %s)\n",
				i+1, rr.Name, rr.Type, rr.Data, rr.TTL)
		}
	}

	// Test 4: Delete a record
	fmt.Println("\n4. Testing DeleteRecords (delete a record)...")
	deleteRecords := []libdns.Record{
		libdns.Address{
			Name: "api",
			IP:   ip3,
			TTL:  1200 * time.Second,
		},
	}

	deletedRecords, err := provider.DeleteRecords(ctx, zone, deleteRecords)
	if err != nil {
		log.Printf("DeleteRecords error: %v", err)
	} else {
		fmt.Printf("✅ %d records deleted\n", len(deletedRecords))
	}

	// Test 5: Test with different record types
	fmt.Println("\n5. Testing with different record types...")

	// CNAME record
	cnameRecord := libdns.CNAME{
		Name:   "www2",
		Target: "www.example.com",
		TTL:    300 * time.Second,
	}

	// MX record
	mxRecord := libdns.MX{
		Name:       "@",
		Preference: 10,
		Target:     "mail.example.com",
		TTL:        600 * time.Second,
	}

	// NS record
	nsRecord := libdns.NS{
		Name:   "@",
		Target: "ns1.example.com",
		TTL:    86400 * time.Second,
	}

	mixedRecords := []libdns.Record{cnameRecord, mxRecord, nsRecord}

	addedMixedRecords, err := provider.AppendRecords(ctx, zone, mixedRecords)
	if err != nil {
		log.Printf("AppendRecords error (mixed types): %v", err)
	} else {
		fmt.Printf("✅ %d mixed records added\n", len(addedMixedRecords))
		for i, record := range addedMixedRecords {
			rr := record.RR()
			fmt.Printf("  %d. %s %s %s (TTL: %s)\n",
				i+1, rr.Name, rr.Type, rr.Data, rr.TTL)
		}
	}

	fmt.Println("\n=== Test completed ===")
}
