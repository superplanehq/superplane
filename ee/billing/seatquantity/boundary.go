// Package seatquantity holds the rules for an explicit next-renewal seat
// change. It does not call a provider and it does not reduce seats by itself.
package seatquantity

import "time"

// MutationGuard is the time before renewal when a seat change cannot be sent.
// Schedule, change, cancel, and purchase registration and final dispatch all
// use it. The purchase confirmation window and the clock-skew margin add up
// to this guard.
const MutationGuard = 10 * time.Minute

// AllowsMutation reports that a mutation sent at now still lands before renewal.
// Exactly MutationGuard remaining is denied. A stale confirmation does not
// move to a later billing period.
func AllowsMutation(now, renewal time.Time) bool {
	if renewal.IsZero() {
		return false
	}
	return now.Add(MutationGuard).Before(renewal)
}

// Cutoff is the last instant at which a mutation is denied. A mutation is
// allowed only while now is strictly before this instant.
func Cutoff(renewal time.Time) time.Time {
	return renewal.Add(-MutationGuard)
}

// MemberFloor is the lowest next-period seat count an Admin may schedule.
// The owner counts. Service accounts and pending invitations do not.
func MemberFloor(humanMembers int) int {
	if humanMembers < 1 {
		return 1
	}
	return humanMembers
}
