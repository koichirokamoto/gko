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

type goonOperation string

var (
	put goonOperation = "Put"
	get goonOperation = "Get"
)

// GoonRetryHelper implement retry interface.
type GoonRetryHelper struct {
	*goon.Goon
	v  []interface{}
	op goonOperation
	m  map[int]bool
	f  func(*GoonRetryHelper) error
}

// NewPutGoonRetryHelper return goon retry helper that be set put operation.
func NewPutGoonRetryHelper(g *goon.Goon, v ...interface{}) *GoonRetryHelper {
	return &GoonRetryHelper{
		Goon: g,
		v:    v,
		op:   put,
		m:    makeErrMap(len(v)),
		f:    (*GoonRetryHelper).put,
	}
}

// NewGetGoonRetryHelper return goon retry helper that be set get operation.
func NewGetGoonRetryHelper(g *goon.Goon, v ...interface{}) *GoonRetryHelper {
	return &GoonRetryHelper{
		Goon: g,
		v:    v,
		op:   get,
		m:    makeErrMap(len(v)),
		f:    (*GoonRetryHelper).get,
	}
}

// SwitchPutHelper switch operation to put.
func (g *GoonRetryHelper) SwitchPutHelper() {
	g.op = put
}

// SwitchGetHelper switch operation to get.
func (g *GoonRetryHelper) SwitchGetHelper() {
	g.op = get
}

// DoRetry is implementation retry interface.
func (g *GoonRetryHelper) DoRetry() error {
	return g.f(g)
}

// HandleError is implementation retry interface.
func (g *GoonRetryHelper) HandleError(err error) bool {
	// if err is known datastore error, return false.
	switch err {
	case datastore.ErrConcurrentTransaction:
		return false
	case datastore.ErrInvalidEntityType:
		return false
	case datastore.ErrInvalidKey:
		return false
	case datastore.ErrNoSuchEntity:
		return false
	}
	return true
}

func makeErrMap(i int) map[int]bool {
	m := make(map[int]bool, i)
	for j := 0; j < i; j++ {
		m[j] = true
	}
	return m
}

func (g *GoonRetryHelper) put() error {
	for i, v := range g.v {
		if g.m[i] {
			_, err := g.Put(v)
			if err != nil {
				g.m[i] = true
				return err
			}
			g.m[i] = false
		}
	}
	return nil
}

func (g *GoonRetryHelper) get() error {
	for i, v := range g.v {
		if g.m[i] {
			err := g.Get(v)
			if err != nil {
				g.m[i] = true
				return err
			}
			g.m[i] = false
		}
	}
	return nil
}
