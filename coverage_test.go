package cryptures

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// operation is one API operation from testdata/operations.txt, which lists
// every operation in the published Cryptures OpenAPI document as
// "METHOD /path/{param} operationId".
type operation struct {
	method, template, id string
	re                   *regexp.Regexp
	params               int
}

func loadOperations(t *testing.T) []operation {
	t.Helper()
	f, err := os.Open("testdata/operations.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var ops []operation
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 {
			continue
		}
		pattern := regexp.QuoteMeta(fields[1])
		pattern = regexp.MustCompile(`\\\{[^/]+?\\\}`).ReplaceAllString(pattern, `[^/]+`)
		ops = append(ops, operation{
			method:   fields[0],
			template: fields[1],
			id:       fields[2],
			re:       regexp.MustCompile("^" + pattern + "$"),
			params:   strings.Count(fields[1], "{"),
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return ops
}

// TestEveryOperationIsCovered proves that every operation in the API is
// implemented and exercised by at least one endpoint test. Each tested
// request is attributed to the most specific matching path template (so
// /block/TRON/latest counts for block.latest, not block.get).
func TestEveryOperationIsCovered(t *testing.T) {
	ops := loadOperations(t)
	if len(ops) != 80 {
		t.Fatalf("expected 80 operations in testdata/operations.txt, found %d", len(ops))
	}

	type hit struct{ method, path string }
	var hits []hit
	tables := [][]endpointCase{
		blockchainDataCases, blockchainOperationsCases, blockchainWalletCases, blockchainContractsCases,
		blockchainFeeCases, blockchainLookupsCases, blockchainNFTCases, cardCases, complianceCases,
	}
	for _, table := range tables {
		for _, tc := range table {
			hits = append(hits, hit{tc.method, tc.path})
		}
	}
	// Operations tested outside the JSON tables (multipart upload and binary
	// downloads); see TestBlockchainStorageUploadIPFS and
	// TestComplianceBinaryEndpoints, which assert these exact requests.
	hits = append(hits,
		hit{"POST", "/api/v1/blockchain/storage/ipfs"},
		hit{"GET", "/api/v1/compliance/sessions/a1b2c3d4/documents/front_image"},
		hit{"GET", "/api/v1/compliance/sessions/a1b2c3d4/report"},
	)

	covered := map[string]bool{}
	for _, h := range hits {
		var best *operation
		for i := range ops {
			op := &ops[i]
			if op.method == h.method && op.re.MatchString(h.path) && (best == nil || op.params < best.params) {
				best = op
			}
		}
		if best == nil {
			t.Errorf("tested request %s %s matches no documented operation", h.method, h.path)
			continue
		}
		covered[best.id] = true
	}

	var missing []string
	for _, op := range ops {
		if !covered[op.id] {
			missing = append(missing, op.id+" ("+op.method+" "+op.template+")")
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d operations have no endpoint test:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
	t.Logf("%d/%d operations covered", len(covered), len(ops))
}
