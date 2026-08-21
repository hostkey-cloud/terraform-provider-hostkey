package invapi

import (
	"fmt"
	"strings"
	"sync"
)

// Process-local claims prevent concurrent WaitForPendingServer / LookupPendingServer
// waiters in the same terraform apply from adopting the same newcomer server id
// when hostname/callback correlation is still lagging.
//
// Claims are keyed by owner (typically "invoice:<n>") so the same pending Create
// can re-resolve after a failed link attempt, while another waiter cannot steal
// the id. Successful links keep the claim until process exit (or explicit
// Release) so other waiters with stale pre-order known snapshots never treat
// that id as a free newcomer again.
var (
	pendingClaimMu sync.Mutex
	pendingClaims  = map[int]string{} // server id → owner
)

func pendingClaimOwner(invoice int, callback string) string {
	if invoice > 0 {
		return fmt.Sprintf("invoice:%d", invoice)
	}
	callback = strings.TrimSpace(callback)
	if callback != "" {
		return "callback:" + callback
	}
	return ""
}

// tryClaimPendingServerID records id for owner. Returns true if the claim is
// newly taken or already held by the same owner; false if another owner holds it.
func tryClaimPendingServerID(id int, owner string) bool {
	if id <= 0 || owner == "" {
		return false
	}
	pendingClaimMu.Lock()
	defer pendingClaimMu.Unlock()
	if cur, ok := pendingClaims[id]; ok {
		return cur == owner
	}
	pendingClaims[id] = owner
	return true
}

// ClaimPendingServerID is the exported form of tryClaimPendingServerID for
// Create paths that receive a server id directly from order_instance.
func ClaimPendingServerID(id int, owner string) bool {
	return tryClaimPendingServerID(id, owner)
}

// ReleasePendingServerClaim drops a claim when Create fails before the real
// server id is written to Terraform state. Only the owning waiter may release.
func ReleasePendingServerClaim(id int, owner string) {
	if id <= 0 || owner == "" {
		return
	}
	pendingClaimMu.Lock()
	defer pendingClaimMu.Unlock()
	if cur, ok := pendingClaims[id]; ok && cur == owner {
		delete(pendingClaims, id)
	}
}

// PendingClaimOwner builds the claim owner string for an invoice/callback pair.
func PendingClaimOwner(invoice int, callback string) string {
	return pendingClaimOwner(invoice, callback)
}

func claimOwnerOf(id int) (string, bool) {
	pendingClaimMu.Lock()
	defer pendingClaimMu.Unlock()
	owner, ok := pendingClaims[id]
	return owner, ok
}

// availableNewcomerIDs returns ids not in known and not claimed by another owner.
// Ids already claimed by owner are included so the same waiter can re-link.
func availableNewcomerIDs(known map[int]struct{}, ids []int, owner string) []int {
	out := make([]int, 0, len(ids))
	pendingClaimMu.Lock()
	defer pendingClaimMu.Unlock()
	for _, id := range ids {
		if _, ok := known[id]; ok {
			continue
		}
		if cur, ok := pendingClaims[id]; ok && cur != owner {
			continue
		}
		out = append(out, id)
	}
	return out
}

// resetPendingServerClaimsForTest clears the process-local claim map between tests.
func resetPendingServerClaimsForTest() {
	pendingClaimMu.Lock()
	defer pendingClaimMu.Unlock()
	pendingClaims = map[int]string{}
}
