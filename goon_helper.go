package gko

import (
	"github.com/mjibson/goon"
	"google.golang.org/appengine/datastore"
)

// GoonHelper is goon helper.
type GoonHelper struct {
	*goon.Goon
	i     *goon.Iterator
	q     *datastore.Query
	limit int
	ret   int
}

// NewGoonHelper return new goon helper.
func NewGoonHelper(g *goon.Goon, kind string) *GoonHelper {
	return &GoonHelper{g, nil, datastore.NewQuery(kind), 0, 0}
}

// Filter set filter to datastore query.
func (g *GoonHelper) Filter(field string, value interface{}) *GoonHelper {
	g.q = g.q.Filter(field, value)
	return g
}

// OrderAsc set ascending order to datastore query.
func (g *GoonHelper) OrderAsc(field string) *GoonHelper {
	g.q = g.q.Order(field)
	return g
}

// OrderDesc set descending order to datastore query.
func (g *GoonHelper) OrderDesc(field string) *GoonHelper {
	g.q = g.q.Order("-" + field)
	return g
}

// Limit set limit to datastore query.
func (g *GoonHelper) Limit(limit int) *GoonHelper {
	g.limit = limit
	g.q = g.q.Limit(limit)
	return g
}

// Start set cursor to datastore query.
func (g *GoonHelper) Start(c datastore.Cursor) *GoonHelper {
	g.q = g.q.Start(c)
	return g
}

// Count return total entity count corresponded to query.
func (g *GoonHelper) Count() (int, error) {
	return g.q.Count(g.Context)
}

// RunQuery run datastore query.
func (g *GoonHelper) RunQuery() *GoonHelper {
	g.i = g.Run(g.q)
	return g
}

// GetResult return iterated result.
//
// if key is non-nil, add one to ret.
func (g *GoonHelper) GetResult(dst interface{}) (*datastore.Key, error) {
	key, err := g.i.Next(dst)
	if key != nil {
		g.ret++
	}
	return key, err
}

// HasNext return true if there is more entity in datastore than limit.
func (g *GoonHelper) HasNext() (bool, string, error) {
	if g.ret < g.limit {
		return false, "", nil
	}
	c, err := g.i.Cursor()
	if err != nil {
		// if cursor is empty, there is no more entity in datastore.
		return false, "", nil
	}

	// if number of result is equal to limit, check whether there is at least one entity in datastore.
	i, err := g.Start(c).Limit(1).Count()
	if err != nil {
		return false, "", err
	}
	return i == 1, c.String(), nil
}
