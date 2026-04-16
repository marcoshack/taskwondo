package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock namespace invite repo ---

type mockNamespaceInviteRepo struct {
	invites map[uuid.UUID]*model.NamespaceInvite
	byCode  map[string]*model.NamespaceInvite
}

func newMockNamespaceInviteRepo() *mockNamespaceInviteRepo {
	return &mockNamespaceInviteRepo{
		invites: make(map[uuid.UUID]*model.NamespaceInvite),
		byCode:  make(map[string]*model.NamespaceInvite),
	}
}

func (r *mockNamespaceInviteRepo) Create(_ context.Context, inv *model.NamespaceInvite) error {
	if _, ok := r.byCode[inv.Code]; ok {
		return model.ErrAlreadyExists
	}
	cp := *inv
	r.invites[inv.ID] = &cp
	r.byCode[inv.Code] = &cp
	return nil
}

func (r *mockNamespaceInviteRepo) GetByCode(_ context.Context, code string) (*model.NamespaceInvite, error) {
	inv, ok := r.byCode[code]
	if !ok {
		return nil, model.ErrNotFound
	}
	return inv, nil
}

func (r *mockNamespaceInviteRepo) GetByID(_ context.Context, id uuid.UUID) (*model.NamespaceInvite, error) {
	inv, ok := r.invites[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return inv, nil
}

func (r *mockNamespaceInviteRepo) ListByNamespace(_ context.Context, nsID uuid.UUID) ([]model.NamespaceInvite, error) {
	var out []model.NamespaceInvite
	for _, inv := range r.invites {
		if inv.NamespaceID == nsID {
			out = append(out, *inv)
		}
	}
	return out, nil
}

func (r *mockNamespaceInviteRepo) IncrementUseCount(_ context.Context, id uuid.UUID) error {
	inv, ok := r.invites[id]
	if !ok {
		return model.ErrNotFound
	}
	if inv.MaxUses > 0 && inv.UseCount >= inv.MaxUses {
		return model.ErrNotFound
	}
	inv.UseCount++
	return nil
}

func (r *mockNamespaceInviteRepo) Delete(_ context.Context, id uuid.UUID) error {
	inv, ok := r.invites[id]
	if !ok {
		return model.ErrNotFound
	}
	delete(r.invites, id)
	delete(r.byCode, inv.Code)
	return nil
}

// --- Helpers ---

func newInviteTestService() (*NamespaceService, *mockNamespaceRepo, *mockNamespaceMemberRepo, *mockNamespaceUserRepo, *mockNamespaceInviteRepo) {
	nsRepo := newMockNamespaceRepo()
	memberRepo := newMockNamespaceMemberRepo()
	projectRepo := newMockNamespaceProjectRepo()
	userRepo := newMockNamespaceUserRepo()
	settingsRepo := newMockNamespaceSystemSettingsRepo()
	inviteRepo := newMockNamespaceInviteRepo()
	svc := NewNamespaceService(nsRepo, memberRepo, projectRepo, newMockNamespaceProjectMemberRepo(), userRepo, settingsRepo, nil, inviteRepo)
	return svc, nsRepo, memberRepo, userRepo, inviteRepo
}

func seedNamespace(t *testing.T, nsRepo *mockNamespaceRepo, memberRepo *mockNamespaceMemberRepo, slug string, ownerID uuid.UUID) *model.Namespace {
	t.Helper()
	ns := &model.Namespace{ID: uuid.New(), Slug: slug, DisplayName: slug, CreatedBy: ownerID}
	if err := nsRepo.Create(context.Background(), ns); err != nil {
		t.Fatalf("creating namespace: %v", err)
	}
	if err := memberRepo.Add(context.Background(), &model.NamespaceMember{
		NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner,
	}); err != nil {
		t.Fatalf("adding owner member: %v", err)
	}
	return ns
}

// --- Tests ---

func TestCreateNamespaceEmailInvite_CreatesInviteForUnregisteredEmail(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, inviteRepo := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	result, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, "new@example.com", "member", nil)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result.Invite.InviteeEmail == nil || *result.Invite.InviteeEmail != "new@example.com" {
		t.Fatalf("expected invitee email preserved")
	}
	if result.Invite.MaxUses != 1 {
		t.Fatalf("expected max_uses=1 for email invite, got %d", result.Invite.MaxUses)
	}

	stored, err := inviteRepo.GetByCode(context.Background(), result.Invite.Code)
	if err != nil {
		t.Fatalf("invite not persisted: %v", err)
	}
	if stored.NamespaceID != ns.ID {
		t.Fatalf("invite namespace_id mismatch")
	}
}

func TestCreateNamespaceEmailInvite_CreatesInviteForRegisteredEmail(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, _ := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	// Add a registered user NOT in the namespace
	otherID := userRepo.addUser(model.RoleUser)
	otherEmail := otherID.String() + "@test.com"

	result, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, otherEmail, "member", nil)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// Even for registered users, a pending invite must be created — they are NOT auto-added.
	if _, err := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, otherID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected user not to be auto-added as member, got err=%v", err)
	}
	if result.Invite.InviteeEmail == nil || *result.Invite.InviteeEmail != otherEmail {
		t.Fatalf("expected invite to carry invitee email")
	}
}

func TestCreateNamespaceEmailInvite_RejectsAlreadyMember(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, _ := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	existingID := userRepo.addUser(model.RoleUser)
	existingEmail := existingID.String() + "@test.com"
	_ = memberRepo.Add(context.Background(), &model.NamespaceMember{
		NamespaceID: ns.ID, UserID: existingID, Role: model.NamespaceRoleMember,
	})

	_, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, existingEmail, "admin", nil)
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestCreateNamespaceEmailInvite_RejectsOwnerRole(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, _ := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	_, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, "new@example.com", "owner", nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for owner role, got %v", err)
	}
}

func TestCreateNamespaceEmailInvite_RequiresAdminOrOwner(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, _ := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	memberID := userRepo.addUser(model.RoleUser)
	_ = memberRepo.Add(context.Background(), &model.NamespaceMember{
		NamespaceID: ns.ID, UserID: memberID, Role: model.NamespaceRoleMember,
	})

	_, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(memberID), ns.Slug, "new@example.com", "member", nil)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-admin, got %v", err)
	}
}

func TestAcceptNamespaceInvite_AddsUserAsMember(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, inviteRepo := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	inviteeID := userRepo.addUser(model.RoleUser)
	inviteeEmail := inviteeID.String() + "@test.com"

	createResult, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, inviteeEmail, "admin", nil)
	if err != nil {
		t.Fatalf("creating invite: %v", err)
	}

	result, err := svc.AcceptNamespaceInvite(context.Background(), nsUserAuthInfo(inviteeID), createResult.Invite.Code)
	if err != nil {
		t.Fatalf("accepting invite: %v", err)
	}
	if result.Namespace.ID != ns.ID {
		t.Fatalf("expected namespace %s, got %s", ns.ID, result.Namespace.ID)
	}
	if result.RoleNotApplied {
		t.Fatalf("expected role to be applied for new member")
	}

	member, err := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, inviteeID)
	if err != nil {
		t.Fatalf("member not added: %v", err)
	}
	if member.Role != model.NamespaceRoleAdmin {
		t.Fatalf("expected admin role, got %s", member.Role)
	}

	// Invite use count incremented
	stored, _ := inviteRepo.GetByID(context.Background(), createResult.Invite.ID)
	if stored.UseCount != 1 {
		t.Fatalf("expected UseCount=1, got %d", stored.UseCount)
	}
}

func TestAcceptNamespaceInvite_ExpiredRejected(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, inviteRepo := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	expired := time.Now().Add(-time.Hour)
	inv := &model.NamespaceInvite{
		ID: uuid.New(), NamespaceID: ns.ID, Code: "expired01", Role: "member",
		CreatedBy: ownerID, ExpiresAt: &expired,
	}
	_ = inviteRepo.Create(context.Background(), inv)

	inviteeID := userRepo.addUser(model.RoleUser)

	_, err := svc.AcceptNamespaceInvite(context.Background(), nsUserAuthInfo(inviteeID), inv.Code)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for expired invite, got %v", err)
	}
}

func TestAcceptNamespaceInvite_UpgradesRoleWhenHigher(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, _ := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	existingID := userRepo.addUser(model.RoleUser)
	existingEmail := existingID.String() + "@test.com"
	_ = memberRepo.Add(context.Background(), &model.NamespaceMember{
		NamespaceID: ns.ID, UserID: existingID, Role: model.NamespaceRoleMember,
	})

	// Remove the member temporarily so we can create the invite, then re-add
	_ = memberRepo.Remove(context.Background(), ns.ID, existingID)
	createResult, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, existingEmail, "admin", nil)
	if err != nil {
		t.Fatalf("creating invite: %v", err)
	}
	_ = memberRepo.Add(context.Background(), &model.NamespaceMember{
		NamespaceID: ns.ID, UserID: existingID, Role: model.NamespaceRoleMember,
	})

	result, err := svc.AcceptNamespaceInvite(context.Background(), nsUserAuthInfo(existingID), createResult.Invite.Code)
	if err != nil {
		t.Fatalf("accepting invite: %v", err)
	}
	if result.RoleNotApplied {
		t.Fatalf("expected upgrade to be applied")
	}

	member, _ := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, existingID)
	if member.Role != model.NamespaceRoleAdmin {
		t.Fatalf("expected admin after upgrade, got %s", member.Role)
	}
}

func TestAcceptNamespaceInvite_DoesNotDowngrade(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, inviteRepo := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	existingID := userRepo.addUser(model.RoleUser)
	_ = memberRepo.Add(context.Background(), &model.NamespaceMember{
		NamespaceID: ns.ID, UserID: existingID, Role: model.NamespaceRoleAdmin,
	})

	// Craft a member-role invite directly
	inv := &model.NamespaceInvite{
		ID: uuid.New(), NamespaceID: ns.ID, Code: "lowrole0", Role: "member",
		CreatedBy: ownerID, MaxUses: 1,
	}
	_ = inviteRepo.Create(context.Background(), inv)

	result, err := svc.AcceptNamespaceInvite(context.Background(), nsUserAuthInfo(existingID), inv.Code)
	if err != nil {
		t.Fatalf("accepting invite: %v", err)
	}
	if !result.RoleNotApplied {
		t.Fatalf("expected RoleNotApplied=true")
	}
	if result.ExistingRole != model.NamespaceRoleAdmin || result.InviteRole != model.NamespaceRoleMember {
		t.Fatalf("unexpected role fields: existing=%s invite=%s", result.ExistingRole, result.InviteRole)
	}

	member, _ := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, existingID)
	if member.Role != model.NamespaceRoleAdmin {
		t.Fatalf("member role should not be downgraded, got %s", member.Role)
	}
}

func TestGetNamespaceInviteInfo_ReturnsPublicInfo(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, _ := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	createResult, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, "new@example.com", "member", nil)
	if err != nil {
		t.Fatalf("creating invite: %v", err)
	}

	info, err := svc.GetNamespaceInviteInfo(context.Background(), createResult.Invite.Code)
	if err != nil {
		t.Fatalf("expected info, got %v", err)
	}
	if info.NamespaceSlug != ns.Slug {
		t.Fatalf("unexpected slug: %s", info.NamespaceSlug)
	}
	if info.Role != "member" {
		t.Fatalf("unexpected role: %s", info.Role)
	}
	if info.Expired || info.Full {
		t.Fatalf("invite should not be expired or full")
	}
}

func TestDeleteNamespaceInvite_RemovesInvite(t *testing.T) {
	svc, nsRepo, memberRepo, userRepo, inviteRepo := newInviteTestService()
	ownerID := userRepo.addUser(model.RoleUser)
	ns := seedNamespace(t, nsRepo, memberRepo, "acme", ownerID)

	createResult, err := svc.CreateNamespaceEmailInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, "new@example.com", "member", nil)
	if err != nil {
		t.Fatalf("creating invite: %v", err)
	}

	if err := svc.DeleteNamespaceInvite(context.Background(), nsUserAuthInfo(ownerID), ns.Slug, createResult.Invite.ID); err != nil {
		t.Fatalf("deleting invite: %v", err)
	}

	if _, err := inviteRepo.GetByID(context.Background(), createResult.Invite.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("invite should be deleted, got %v", err)
	}
}
