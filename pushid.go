//
// Fancy ID generator that creates 20-character string identifiers with the following properties:
//
// 1. They're based on timestamp so that they sort *after* any existing ids.
// 2. They contain 72-bits of random data after the timestamp so that IDs won't collide with other clients' IDs.
// 3. They sort *lexicographically* (so the timestamp is converted to characters that will sort properly).
// 4. They're monotonically increasing. Even if you generate more than one in the same timestamp, the
// latter ones will sort after the former ones. We do this by using the previous random bits
// but "incrementing" them by 1 (only in the case of a timestamp collision).
//
// Adapted from:
// * https://www.firebase.com/blog/2015-02-11-firebase-unique-identifiers.html
// * https://gist.github.com/cabrel/4e085a9de3632d788fd4 (forked for retention, original: https://gist.github.com/themartorana/8c8b704432c8be1fed9a)
//
package pushid

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

var (
	// Timestamp of last push, used to prevent local collisions if you push twice in one ms.
	lastPushTime int64

	// We generate 72-bits of randomness which get turned into 12 characters and appended to the
	// timestamp to prevent collisions with other clients. We store the last characters we
	// generated because in the event of a collision, we'll use those same characters except
	// "incremented" by one.
	lastRandChars []int8
)

const (
	// Modeled after base64 web-safe chars, but ordered by ASCII.
	PUSH_CHARS string = "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
)

func init() {
	lastRandChars = make([]int8, 12, 12)
}

// Generate returns a best-effort unique push id.
//
// Taken from: https://www.firebase.com/blog/2015-02-11-firebase-unique-identifiers.html
//
// >  A push ID contains 120 bits of information. The first 48 bits are a timestamp, which both reduces the chance of
// >  collision and allows consecutively created push IDs to sort chronologically. The timestamp is followed by 72 bits
// >  of randomness, which ensures that even two people creating push IDs at the exact same millisecond are extremely
// >  unlikely to generate identical IDs. One caveat to the randomness is that in order to preserve chronological
// >  ordering if a client creates multiple push IDs in the same millisecond, we just ‘increment’ the random bits
// >  by one.
// >
// >  To turn our 120 bits of information (timestamp + randomness) into an ID that can be used as a Firebase key,
// >  we basically base64 encode it into ASCII characters, but we use a modified base64 alphabet that ensures the
// >  IDs will still sort correctly when ordered lexicographically (since Firebase keys are ordered lexicographically).
func Generate() (string, error) {
	now := time.Now().UTC().UnixNano() / 1000000
	duplicateTime := now == lastPushTime
	lastPushTime = now

	timeStampChars := make([]string, 8, 8)
	for i := 7; i >= 0; i-- {
		pcIndex := int64(math.Mod(float64(now), 64.0))
		timeStampChars[i] = string(PUSH_CHARS[pcIndex])
		now = int64(math.Floor(float64(now) / 64.0))
	}

	if now != 0 {
		return "", fmt.Errorf("We should have converted the entire timestamp.")
	}

	id := strings.Join(timeStampChars, "")

	if !duplicateTime {
		for i := 0; i < 12; i++ {
			lastRandChars[i] = int8(math.Floor(rand.Float64() * 64.0))
		}
	} else {
		var i int
		for i = 11; i >= 0 && lastRandChars[i] == 63; i-- {
			lastRandChars[i] = 0
		}

		lastRandChars[i]++
	}

	for i := 0; i < 12; i++ {
		id = fmt.Sprintf("%s%s", id, string(PUSH_CHARS[lastRandChars[i]]))
	}

	if len(id) != 20 {
		return "", fmt.Errorf("Length should be 20")
	}

	return id, nil
}
