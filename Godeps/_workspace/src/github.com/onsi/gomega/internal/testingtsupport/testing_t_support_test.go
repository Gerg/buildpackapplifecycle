package testingtsupport_test

import (
	. "github.com/cloudfoundry-incubator/buildpack_app_lifecycle/Godeps/_workspace/src/github.com/onsi/gomega"
	"testing"
)

func TestTestingT(t *testing.T) {
	RegisterTestingT(t)
	Ω(true).Should(BeTrue())
}
