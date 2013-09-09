package bolt

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/boltdb/bolt"
)

const FragSeparator = ':'
const PathSeparator = '/'

func SplitBuckets(quoted string) ([][]byte, error) {
	var result [][]byte
	for _, q := range strings.Split(quoted, string(PathSeparator)) {
		k, err := DecodeKey(q)
		if err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, nil
}

func LookupBucket(tx *bolt.Tx, buckets [][]byte) (*bolt.Bucket, error) {
	if len(buckets) == 0 {
		return nil, errors.New("empty bucket path")
	}
	b := tx.Bucket(buckets[0])
	if b == nil {
		return nil, errors.New("bucket not found")
	}
	for _, name := range buckets[1:] {
		b = b.Bucket(name)
		if b == nil {
			return nil, errors.New("bucket not found")
		}
	}
	return b, nil
}

func DecodeKey(quoted string) ([]byte, error) {
	var key []byte
	for _, frag := range strings.Split(quoted, string(FragSeparator)) {
		if frag == "" {
			return nil, fmt.Errorf("quoted key cannot have empty fragment: %s", quoted)
		}
		switch {
		case strings.HasPrefix(frag, "@"):
			f, err := hex.DecodeString(frag[1:])
			if err != nil {
				return nil, err
			}
			key = append(key, f...)
		default:
			key = append(key, frag...)
		}
	}
	return key, nil
}

func isSafe(r rune) bool {
	if r > unicode.MaxASCII {
		return false
	}
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return true
	}
	switch r {
	case FragSeparator:
		return false
	case '.', ',', '-':
		return true
	}
	return false
}

const prettyTheshold = 2

func EncodeKey(key []byte) string {
	// we do sloppy work and process safe bytes only at the beginning
	// and end; this avoids many false positives in large binary data

	var left, middle, right string

	{
		mid := bytes.TrimLeftFunc(key, isSafe)
		if len(key)-len(mid) > prettyTheshold {
			left = string(key[:len(key)-len(mid)]) + string(FragSeparator)
			key = mid
		}
	}

	{
		mid := bytes.TrimRightFunc(key, isSafe)
		if len(key)-len(mid) > prettyTheshold {
			right = string(FragSeparator) + string(key[len(mid):])
			key = mid
		}
	}

	if len(key) > 0 {
		middle = "@" + hex.EncodeToString(key)
	}

	return strings.Trim(left+middle+right, string(FragSeparator))
}
