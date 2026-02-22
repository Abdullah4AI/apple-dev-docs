package asc

import "testing"

func TestFinanceRegions(t *testing.T) {
	regions := FinanceRegions()
	if len(regions) == 0 {
		t.Fatal("expected finance regions to be populated")
	}

	var hasUS, hasZ1, hasZZ bool
	for _, region := range regions {
		switch region.RegionCode {
		case "US":
			hasUS = true
		case "Z1":
			hasZ1 = true
		case "ZZ":
			hasZZ = true
		}
	}

	if !hasUS || !hasZ1 || !hasZZ {
		t.Fatalf("expected regions to include US, Z1, and ZZ (got US=%t Z1=%t ZZ=%t)", hasUS, hasZ1, hasZZ)
	}
}
