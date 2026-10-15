package identifier_test

import (
	"fmt"

	identifier "github.com/faustbrian/go-identifier/v2"
)

func ExampleNormalize() {
	profile, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{
		ID: "account.username", Version: 1,
		Unicode:  identifier.UnicodeNormalizationNFKC,
		CaseFold: true, TrimSpace: true, MaximumBytes: 128,
	})
	if err != nil {
		panic(err)
	}
	input, err := identifier.NewInput("  Straße  ")
	if err != nil {
		panic(err)
	}
	canonical, err := identifier.Normalize(input, profile)
	if err != nil {
		panic(err)
	}
	// Persistence deliberately receives every field, not merely the lookup string.
	record := canonical.Record()
	restored, err := identifier.RestoreCanonical(record, profile)
	if err != nil {
		panic(err)
	}
	fmt.Println(restored.Equal(canonical))
	fmt.Println(record.ProfileID, record.ProfileVersion)
	// Output:
	// true
	// account.username 1
}
