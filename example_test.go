package identifier_test

import (
	"fmt"

	identifieruuid "github.com/faustbrian/go-identifier/uuid"
)

func Example() {
	generator := identifieruuid.NewV7Generator(nil, nil)

	id, err := generator.New()
	if err != nil {
		panic(err)
	}

	fmt.Printf("uuid version: %d\n", id.Version())
	// Output: uuid version: 7
}
