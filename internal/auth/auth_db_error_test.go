package auth

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestLegacyOrgTeamHelpersDoNotFatalOnConnectError(t *testing.T) {
	connectErr := errors.New("connect failed")
	stubAuthDatabase(t, nil, connectErr)
	silenceDefaultLog(t)

	assertNoPanic(t, "CreateOrganizationRecord", func() {
		CreateOrganizationRecord("org")
	})
	assertNoPanic(t, "UpdateOrganizationRecord", func() {
		UpdateOrganizationRecord("org")
	})
	assertNoPanic(t, "DeleteOrganizationRecord", func() {
		DeleteOrganizationRecord("org")
	})
	assertNoPanic(t, "CreateTeamRecord", func() {
		CreateTeamRecord("team", "org")
	})
	assertNoPanic(t, "UpdateTeamRecord", func() {
		UpdateTeamRecord("team", "org")
	})
	assertNoPanic(t, "DeleteTeamRecord", func() {
		DeleteTeamRecord("team")
	})

	if got := GetOrganizationRecord("org"); got != nil {
		t.Fatalf("expected nil organization on connect error, got %#v", got)
	}
	if got := GetTeamRecord("team"); got != nil {
		t.Fatalf("expected nil team on connect error, got %#v", got)
	}
}

func TestGetRecordWithErrorReturnsNilOnRecordNotFound(t *testing.T) {
	db := newAuthTestDB(t, gorm.ErrRecordNotFound)

	organization, err := getOrganizationRecord(db, "missing-org")
	if err != nil {
		t.Fatalf("expected no organization error on not found, got %v", err)
	}
	if organization != nil {
		t.Fatalf("expected nil organization on not found, got %#v", organization)
	}

	team, err := getTeamRecord(db, "missing-team")
	if err != nil {
		t.Fatalf("expected no team error on not found, got %v", err)
	}
	if team != nil {
		t.Fatalf("expected nil team on not found, got %#v", team)
	}
}

func TestCreateKeyRecordWithErrorReturnsLookupError(t *testing.T) {
	queryErr := errors.New("team lookup failed")
	db := newAuthTestDB(t, queryErr)
	stubAuthDatabase(t, db, nil)

	err := CreateKeyRecordWithError("sk-test", "test", []string{"*"}, "team", "org", 1, 60, 0)
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected lookup error %v, got %v", queryErr, err)
	}
}

func TestCreateKeyRecordDoesNotFatalOnLookupError(t *testing.T) {
	queryErr := errors.New("team lookup failed")
	db := newAuthTestDB(t, queryErr)
	stubAuthDatabase(t, db, nil)
	silenceDefaultLog(t)

	assertNoPanic(t, "CreateKeyRecord", func() {
		CreateKeyRecord("sk-test", "test", []string{"*"}, "team", "org", 1, 60, 0)
	})
}

func stubAuthDatabase(t *testing.T, db *gorm.DB, err error) {
	t.Helper()

	oldConnect := connectAuthDatabase
	oldClose := closeAuthDatabaseConnection
	connectAuthDatabase = func() (*gorm.DB, error) {
		return db, err
	}
	closeAuthDatabaseConnection = func(*gorm.DB) {}

	t.Cleanup(func() {
		connectAuthDatabase = oldConnect
		closeAuthDatabaseConnection = oldClose
	})
}

func newAuthTestDB(t *testing.T, err error) *gorm.DB {
	t.Helper()

	db, openErr := gorm.Open(postgres.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if openErr != nil {
		t.Fatalf("open dry test database handle: %v", openErr)
	}
	if err != nil {
		db.AddError(err)
	}
	return db
}

func assertNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func silenceDefaultLog(t *testing.T) {
	t.Helper()

	oldOutput := log.Writer()
	log.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() {
		log.SetOutput(oldOutput)
	})
}
