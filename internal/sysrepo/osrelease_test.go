package sysrepo

import "testing"

func TestIDMatches(t *testing.T) {
	if !idMatches("ubuntu", "debian", "debian", "ubuntu") {
		t.Fatal("ubuntu should match debian id_like")
	}
	if !idMatches("rocky", "rhel fedora", "rhel") {
		t.Fatal("rocky should match rhel id_like")
	}
	if idMatches("ubuntu", "debian", "fedora") {
		t.Fatal("ubuntu should not match fedora only")
	}
	if !idMatches("debian", "", "debian") {
		t.Fatal("debian id alone")
	}
}
