package testredis

import "github.com/alicebob/miniredis/v2"

func CreateMiniRedis() (*miniredis.Miniredis, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, err
	}
	return mr, err
}
