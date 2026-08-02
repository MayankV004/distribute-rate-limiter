package tier

// TODO (Phase 4): implement StaticResolver.
//
//   type StaticResolver struct {
//       keys         map[string]string   // identity → tier name
//       defaultTier  string              // returned when identity not in map
//       unknownPolicy string             // "deny" | "default_tier"
//   }
//
//   func NewStatic(keys map[string]string, defaultTier, unknownPolicy string) *StaticResolver
//
//   func (r *StaticResolver) Resolve(_ context.Context, identity string) (string, error)
//     - Lookup identity in r.keys
//     - If found → return tier name, nil
//     - If not found AND unknownPolicy == "deny" → return "", ErrUnknown
//     - If not found AND unknownPolicy == "default_tier" → return r.defaultTier, nil
//
// Config source (loaded in Phase 4):
//   api_keys:
//     "key_free_abc123": free
//     "key_pro_xyz789":  pro
//   identity:
//     default_tier: free
//     unknown_key_policy: deny   # safe default — unrecognised key gets no quota
