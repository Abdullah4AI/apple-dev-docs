package distribute

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/signing"
)

func TestValidateDistributionReconcileEvidenceBindsExactReceiptAndProfile(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*distributionEvidenceFixture)
	}{
		{name: "exact"},
		{name: "different plan", mutate: func(f *distributionEvidenceFixture) { f.reconcile.PlanHash = strings.Repeat("9", 64) }},
		{name: "incomplete receipt", mutate: func(f *distributionEvidenceFixture) { f.reconcile.Complete = false }},
		{name: "different profile resource", mutate: func(f *distributionEvidenceFixture) { f.reconcile.Actions[0].ResourceID = "different-profile" }},
		{name: "different profile output", mutate: func(f *distributionEvidenceFixture) { f.reconcile.Actions[0].OutputPath += ".different" }},
		{name: "failed action", mutate: func(f *distributionEvidenceFixture) { f.reconcile.Actions[0].Status = "failed" }},
		{name: "verified profile digest drift", mutate: func(f *distributionEvidenceFixture) { f.verified.MainProfile.SHA256 = strings.Repeat("9", 64) }},
		{name: "run-local profile digest drift", mutate: func(f *distributionEvidenceFixture) { f.profile.SHA256 = strings.Repeat("9", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDistributionEvidenceFixture(t)
			if test.mutate != nil {
				test.mutate(&fixture)
			}
			fixture.publishReconcileEvidence(t)
			err := validateDistributionReconcileEvidence(fixture.stateDir, fixture.plan, fixture.state, fixture.verified, fixture.receipt, fixture.profile)
			if test.name == "exact" && err != nil {
				t.Fatalf("validateDistributionReconcileEvidence() error: %v", err)
			}
			if test.name != "exact" && err == nil {
				t.Fatal("mismatched reconcile evidence was accepted")
			}
		})
	}
}

func TestValidateDistributionReconcileEvidenceRejectsUnknownReceiptFields(t *testing.T) {
	fixture := newDistributionEvidenceFixture(t)
	payload, err := json.Marshal(fixture.reconcile)
	if err != nil {
		t.Fatal(err)
	}
	payload = append(payload[:len(payload)-1], []byte(`,"unexpected":"secret"}`)...)
	fixture.receipt.SHA256 = digestDistributionBytes(payload)
	writeDistributionEvidenceFixtureFile(t, fixture.stateDir, fixture.state.RunID, fixture.receipt.Path, payload)
	writeDistributionEvidenceFixtureFile(t, fixture.stateDir, fixture.state.RunID, fixture.profile.Path, fixture.profileData)
	if err := validateDistributionReconcileEvidence(fixture.stateDir, fixture.plan, fixture.state, fixture.verified, fixture.receipt, fixture.profile); err == nil {
		t.Fatal("receipt with an unknown field was accepted")
	}
}

func TestValidateDistributionSigningEvidenceRequiresExactSuccessfulCleanup(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*distributionEvidenceFixture)
	}{
		{name: "removed profile"},
		{name: "reused profile", mutate: func(f *distributionEvidenceFixture) { f.signing.ProfileCleanupState = "reused" }},
		{name: "failed outcome", mutate: func(f *distributionEvidenceFixture) { f.signing.Outcome = "failed" }},
		{name: "child failure", mutate: func(f *distributionEvidenceFixture) { f.signing.ChildExitCode = 23 }},
		{name: "profile cleanup pending", mutate: func(f *distributionEvidenceFixture) { f.signing.ProfileCleanupState = "pending" }},
		{name: "keychain cleanup failed", mutate: func(f *distributionEvidenceFixture) { f.signing.KeychainCleanupState = "failed" }},
		{name: "certificate drift", mutate: func(f *distributionEvidenceFixture) { f.signing.CertificateSHA256 = strings.Repeat("9", 64) }},
		{name: "profile drift", mutate: func(f *distributionEvidenceFixture) { f.signing.ProfileSHA256 = strings.Repeat("9", 64) }},
		{name: "team drift", mutate: func(f *distributionEvidenceFixture) { f.signing.TeamID = "OTHERTEAM" }},
		{name: "future attempt", mutate: func(f *distributionEvidenceFixture) { f.signingReceipt.Path = "signing/receipt-000002.json" }},
		{name: "canonical receipt path", mutate: func(f *distributionEvidenceFixture) { f.signingReceipt.Path = "signing/receipt.json" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDistributionEvidenceFixture(t)
			if test.mutate != nil {
				test.mutate(&fixture)
			}
			payload, err := json.Marshal(fixture.signing)
			if err != nil {
				t.Fatal(err)
			}
			fixture.signingReceipt.SHA256 = digestDistributionBytes(payload)
			writeDistributionEvidenceFixtureFile(t, fixture.stateDir, fixture.state.RunID, fixture.signingReceipt.Path, payload)
			err = validateDistributionSigningEvidence(fixture.stateDir, fixture.plan, fixture.state, fixture.signingReceipt)
			valid := test.name == "removed profile" || test.name == "reused profile"
			if valid && err != nil {
				t.Fatalf("validateDistributionSigningEvidence() error: %v", err)
			}
			if !valid && err == nil {
				t.Fatal("mismatched signing evidence was accepted")
			}
		})
	}
}

func TestValidateDistributionSigningEvidenceRejectsUnknownReceiptFields(t *testing.T) {
	fixture := newDistributionEvidenceFixture(t)
	payload, err := json.Marshal(fixture.signing)
	if err != nil {
		t.Fatal(err)
	}
	payload = append(payload[:len(payload)-1], []byte(`,"unexpected":"secret"}`)...)
	fixture.signingReceipt.SHA256 = digestDistributionBytes(payload)
	writeDistributionEvidenceFixtureFile(t, fixture.stateDir, fixture.state.RunID, fixture.signingReceipt.Path, payload)
	if err := validateDistributionSigningEvidence(fixture.stateDir, fixture.plan, fixture.state, fixture.signingReceipt); err == nil {
		t.Fatal("signing receipt with an unknown field was accepted")
	}
}

type distributionEvidenceFixture struct {
	stateDir       string
	plan           persistedDistributionPlan
	state          persistedDistributionRunState
	reconcile      distributionReconcileReceiptEvidence
	verified       signing.ReconcileReceiptView
	receipt        distributionFileArtifact
	profile        distributionProfileArtifact
	profileData    []byte
	signing        distributionSigningReceiptEvidence
	signingReceipt distributionFileArtifact
}

func newDistributionEvidenceFixture(t *testing.T) distributionEvidenceFixture {
	t.Helper()
	stateDir := filepath.Join(t.TempDir(), "runs")
	reconcileDir := filepath.Join(t.TempDir(), "reconcile")
	plan := validPersistedDistributionPlan(t)
	plan.Paths.StateDir = stateDir
	plan.Reconcile.PlanPath = filepath.Join(reconcileDir, "plan.json")
	plan.Reconcile.ReceiptPath = filepath.Join(reconcileDir, "receipt.json")
	if err := sealDistributionPlan(&plan); err != nil {
		t.Fatal(err)
	}
	state := validCompletedDistributionRun(plan)
	state.Attempt = 1
	profileData := []byte("exact-profile-evidence")
	profile := *state.Artifacts.Profile
	profile.SHA256 = digestDistributionBytes(profileData)
	profile.Path = distributionProfileRelative
	state.Artifacts.Profile = &profile
	mainPath := filepath.Join(reconcileDir, "profiles", profile.UUID+".mobileprovision")
	reconcile := distributionReconcileReceiptEvidence{
		SchemaVersion: 1, PlanHash: plan.Reconcile.PlanHash,
		StartedAt: "2026-08-13T08:00:00Z", UpdatedAt: "2026-08-13T08:01:00Z", Complete: true,
		StateDir: reconcileDir, ReceiptPath: plan.Reconcile.ReceiptPath,
		Actions: []distributionReconcileActionReceiptEvidence{{
			ID: "profile:" + plan.Archive.BundleID, Kind: "createProfile", Status: "completed",
			ResourceID: profile.ResourceID, OutputPath: mainPath,
		}},
	}
	verified := signing.ReconcileReceiptView{
		SchemaVersion: 1, PlanHash: plan.Reconcile.PlanHash, Complete: true, ReceiptPath: plan.Reconcile.ReceiptPath,
		MainProfile: &signing.ReconcileProfileView{
			TargetKind: "application", BundleID: plan.Archive.BundleID, ResourceID: profile.ResourceID,
			UUID: profile.UUID, Path: mainPath, SHA256: profile.SHA256,
		},
	}
	verified.Profiles = []signing.ReconcileProfileView{*verified.MainProfile}
	signingEvidence := distributionSigningReceiptEvidence{
		SchemaVersion: 1, Purpose: "release-testing", Outcome: "succeeded", ChildExitCode: 0,
		CertificateSHA256: plan.Identity.CertificateSHA256, ProfileSHA256: profile.SHA256,
		ProfileUUID: profile.UUID, TeamID: plan.Identity.TeamID, BundleID: plan.Archive.BundleID,
		ProfileCleanupState: "removed", KeychainCleanupState: "deleted",
	}
	return distributionEvidenceFixture{
		stateDir: stateDir, plan: plan, state: state, reconcile: reconcile, verified: verified,
		receipt: distributionFileArtifact{Path: distributionReconcileRelative}, profile: profile, profileData: profileData,
		signing: signingEvidence, signingReceipt: distributionFileArtifact{Path: "signing/receipt-000001.json"},
	}
}

func (fixture *distributionEvidenceFixture) publishReconcileEvidence(t *testing.T) {
	t.Helper()
	payload, err := json.Marshal(fixture.reconcile)
	if err != nil {
		t.Fatal(err)
	}
	fixture.receipt.SHA256 = digestDistributionBytes(payload)
	writeDistributionEvidenceFixtureFile(t, fixture.stateDir, fixture.state.RunID, fixture.receipt.Path, payload)
	writeDistributionEvidenceFixtureFile(t, fixture.stateDir, fixture.state.RunID, fixture.profile.Path, fixture.profileData)
}

func writeDistributionEvidenceFixtureFile(t *testing.T, stateDir, runID, relative string, data []byte) {
	t.Helper()
	directory := filepath.Join(stateDir, runID, filepath.Dir(relative))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(stateDir, runID), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(stateDir, runID, filepath.FromSlash(relative))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}
