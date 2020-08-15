package twembed

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

//go:generate go run embed_mk.go

// New returns an instance that implements tailwind.Dist using data embedded in this package.
func New() Dist {
	return Dist{}
}

// func init() {
// 	// package import causes embedded version to be default if no other is specified
// 	if tailwind.DefaultDist == nil {
// 		tailwind.DefaultDist = New()
// 	}
// }

// Dist implements tailwind.Dist
type Dist struct{}

// OpenDist implements the interface and returns embedded data.
func (d Dist) OpenDist(name string) (io.ReadCloser, error) {

	switch name {
	case "base":
		return ioutil.NopCloser(strings.NewReader(twbase())), nil
	case "utilities":
		return ioutil.NopCloser(strings.NewReader(twutilities())), nil
	case "components":
		return ioutil.NopCloser(strings.NewReader(twcomponents())), nil
	}

	return nil, fmt.Errorf("twembed unknown name %q", name)

}

// PurgeKeyMap returns a map of all of the possible keys that can be purged.
func (d Dist) PurgeKeyMap() map[string]struct{} {
	return twPurgeKeyMap
}
