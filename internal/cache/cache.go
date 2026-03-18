package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/ilovelili/ad-engine/internal/domain"
)

const dashboardKey = "ad-engine:dashboard"

type Cache struct {
	pool      *redis.Pool
	available bool
}

func New(addr, password, db string) *Cache {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 5 * time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			if db != "" && db != "0" {
				if _, err := c.Do("SELECT", db); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	cache := &Cache{pool: pool}
	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err == nil {
		cache.available = true
	}

	return cache
}

func (c *Cache) Available() bool {
	return c != nil && c.available
}

func (c *Cache) SetDashboard(snapshot domain.CampaignSnapshot) error {
	if !c.Available() {
		return nil
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal dashboard snapshot: %w", err)
	}
	conn := c.pool.Get()
	defer conn.Close()
	_, err = conn.Do("SETEX", dashboardKey, 30, body)
	return err
}

func (c *Cache) GetDashboard() (*domain.CampaignSnapshot, error) {
	if !c.Available() {
		return nil, nil
	}
	conn := c.pool.Get()
	defer conn.Close()
	body, err := redis.Bytes(conn.Do("GET", dashboardKey))
	if err == redis.ErrNil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var snapshot domain.CampaignSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (c *Cache) Close() error {
	if c == nil || c.pool == nil {
		return nil
	}
	return c.pool.Close()
}
