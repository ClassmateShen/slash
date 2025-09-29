package license

import (
	"context"
	_ "embed"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/yourselfhosted/slash/proto/gen/api/v1"
	storepb "github.com/yourselfhosted/slash/proto/gen/store"
	"github.com/yourselfhosted/slash/server/profile"
	"github.com/yourselfhosted/slash/store"
)

//go:embed slash.public.pem
var slashPublicRSAKey string

type LicenseService struct {
	Profile *profile.Profile
	Store   *store.Store

	cachedSubscription *v1pb.Subscription
}

// NewLicenseService creates a new LicenseService.
func NewLicenseService(profile *profile.Profile, store *store.Store) *LicenseService {
	return &LicenseService{
		Profile:            profile,
		Store:              store,
		cachedSubscription: getSubscriptionForEnterprisePlan(),
	}
}

func (s *LicenseService) LoadSubscription(ctx context.Context) (*v1pb.Subscription, error) {
	// 直接返回 ENTERPRISE 订阅，忽略 license key
	subscription := getSubscriptionForEnterprisePlan()
	s.cachedSubscription = subscription
	return subscription, nil
}

func (s *LicenseService) UpdateSubscription(ctx context.Context, licenseKey string) (*v1pb.Subscription, error) {
	// 保留接口兼容性，但忽略 licenseKey
	if err := s.UpdateLicenseKey(ctx, ""); err != nil {
		return nil, errors.Wrap(err, "failed to update license key")
	}
	return s.LoadSubscription(ctx)
}

func (s *LicenseService) UpdateLicenseKey(ctx context.Context, licenseKey string) error {
	// 仍可写入空 license key（或完全跳过），这里选择清空
	workspaceGeneralSetting, err := s.Store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace general setting")
	}
	workspaceGeneralSetting.LicenseKey = "" // 强制清空
	_, err = s.Store.UpsertWorkspaceSetting(ctx, &storepb.WorkspaceSetting{
		Key: storepb.WorkspaceSettingKey_WORKSPACE_SETTING_GENERAL,
		Value: &storepb.WorkspaceSetting_General{
			General: workspaceGeneralSetting,
		},
	})
	return errors.Wrap(err, "failed to upsert workspace setting")
}

func (s *LicenseService) GetSubscription() *v1pb.Subscription {
	return s.cachedSubscription
}

func (s *LicenseService) IsFeatureEnabled(feature FeatureType) bool {
	// ENTERPRISE 启用所有功能
	return true
}

// 以下类型保留以避免编译错误，但不再使用
type ValidateResult struct {
	Plan        v1pb.PlanType
	ExpiresTime time.Time
	Trial       bool
	Seats       int
	Features    []FeatureType
}

type Claims struct {
	jwt.RegisteredClaims
	Owner    string `json:"owner"`
	Plan     string `json:"plan"`
	Trial    bool   `json:"trial"`
	Seats    int    `json:"seats"`
	Features []string `json:"features"`
}

// 保留但不使用
func validateLicenseKey(licenseKey string) (*ValidateResult, error) {
	return nil, errors.New("license validation disabled in enterprise mode")
}

func parseLicenseKey(licenseKey string) (*Claims, error) {
	return nil, errors.New("license parsing disabled")
}

func getSubscriptionForFreePlan() *v1pb.Subscription {
	return &v1pb.Subscription{
		Plan:             v1pb.PlanType_FREE,
		Seats:            5,
		ShortcutsLimit:   100,
		CollectionsLimit: 5,
		Features:         []string{},
	}
}

// 新增：返回 ENTERPRISE 订阅
func getSubscriptionForEnterprisePlan() *v1pb.Subscription {
	expiresTime := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

	return &v1pb.Subscription{
		Plan:   v1pb.PlanType_ENTERPRISE,
		Seats:  999999,
		// 由于启用了 unlimited features，limit 可设为 0 或极大值
		ShortcutsLimit:   0, // 或 999999，但通常 unlimited 时前端忽略 limit
		CollectionsLimit: 0,
		ExpiresTime:      timestamppb.New(expiresTime),
		Features: []string{
			"ysh.slash.sso",
			"ysh.slash.advanced-analytics",
			"ysh.slash.unlimited-accounts",
			"ysh.slash.unlimited-shortcuts",
			"ysh.slash.unlimited-collections",
			"ysh.slash.custom-branding",
		},
	}
}