package metadata

import "fmt"

func (c *Client) GetEpisodeByCRC32(crc string) (Episode, error) {
	c.Cache.mu.RLock()
	ep, ok := c.Cache.EpisodesByCRC[crc]
	c.Cache.mu.RUnlock()
	if !ok {
		return Episode{}, fmt.Errorf("episode not found for CRC %s", crc)
	}
	return ep, nil
}

func (c *Client) GetArcTitle(arcNumber int) string {
	c.Cache.mu.RLock()
	arc, ok := c.Cache.ArcsByNumber[arcNumber]
	c.Cache.mu.RUnlock()
	if !ok {
		return fmt.Sprintf("Arc %d", arcNumber)
	}
	return arc.Title
}

func (c *Client) GetArcByNumber(arcNumber int) (Arc, error) {
	c.Cache.mu.RLock()
	arc, ok := c.Cache.ArcsByNumber[arcNumber]
	c.Cache.mu.RUnlock()
	if !ok {
		return Arc{}, fmt.Errorf("arc not found for number %d", arcNumber)
	}
	return arc, nil
}

// EpisodesByArc returns all episodes belonging to a specific arc, deduplicated
// by episode number (one entry per episode — the first CRC32 seen wins).
func (c *Client) EpisodesByArc(arcNumber int) []Episode {
	c.Cache.mu.RLock()
	defer c.Cache.mu.RUnlock()
	seen := map[int]bool{}
	var out []Episode
	for _, ep := range c.Cache.EpisodesByCRC {
		if ep.Arc != arcNumber || seen[ep.Episode] {
			continue
		}
		seen[ep.Episode] = true
		out = append(out, ep)
	}
	return out
}

// Episodes returns a snapshot of all episodes, safe for iteration.
func (c *Client) Episodes() map[string]Episode {
	c.Cache.mu.RLock()
	defer c.Cache.mu.RUnlock()
	snapshot := make(map[string]Episode, len(c.Cache.EpisodesByCRC))
	for k, v := range c.Cache.EpisodesByCRC {
		snapshot[k] = v
	}
	return snapshot
}
