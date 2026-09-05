package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	jpoetwatch "github.com/marcbran/jpoet/pkg/watch"
)

type informerKey struct {
	context   string
	gvr       schema.GroupVersionResource
	namespace string
	name      string
}

type watchEntry struct {
	informer cache.SharedIndexInformer
	cancel   context.CancelFunc
	holders  int
}

type pendingEntry struct {
	done     chan struct{}
	informer cache.SharedIndexInformer
	err      error
}

type watchCache struct {
	clients *clientRegistry

	mu      sync.Mutex
	entries map[informerKey]*watchEntry
	pending map[informerKey]*pendingEntry

	changes func(keys []jpoetwatch.InvocationKey)

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	closeOnce sync.Once
}

func newWatchCache(clients *clientRegistry) *watchCache {
	ctx, cancel := context.WithCancel(context.Background())
	return &watchCache{
		clients: clients,
		entries: map[informerKey]*watchEntry{},
		pending: map[informerKey]*pendingEntry{},
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (c *watchCache) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		c.wg.Wait()
	})
	return nil
}

func (c *watchCache) InvocationKey(_ string, args []any) jpoetwatch.InvocationKey {
	if len(args) < 2 {
		return "kubernetes://invalid"
	}
	ctxName, _ := args[0].(string)
	path, _ := args[1].(string)
	gvr, namespace, name, err := parsePath(path)
	if err != nil {
		return jpoetwatch.InvocationKey("kubernetes://invalid/" + path)
	}
	return fromInformerKey(informerKey{context: ctxName, gvr: gvr, namespace: namespace, name: name})
}

func (c *watchCache) SetChanges(changes func(keys []jpoetwatch.InvocationKey)) {
	c.changes = changes
}

func (c *watchCache) Acquire(key jpoetwatch.InvocationKey) (func(), error) {
	if err := c.ctx.Err(); err != nil {
		return nil, err
	}
	target, ok := toInformerKey(key)
	if !ok {
		return func() {}, nil
	}

	listKey := informerKey{context: target.context, gvr: target.gvr, namespace: target.namespace}
	if release := c.hold(listKey); release != nil {
		return release, nil
	}
	if target.name != "" {
		if release := c.hold(target); release != nil {
			return release, nil
		}
	}

	if _, err := c.informerFor(target); err != nil {
		return nil, err
	}
	release := c.hold(target)
	if release == nil {
		return func() {}, nil
	}
	return release, nil
}

func (c *watchCache) hold(key informerKey) func() {
	c.mu.Lock()
	entry, ok := c.entries[key]
	if ok {
		entry.holders++
	}
	c.mu.Unlock()
	if !ok {
		return nil
	}
	return func() { c.release(key) }
}

func (c *watchCache) release(key informerKey) {
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		return
	}
	entry.holders--
	teardown := entry.holders <= 0
	if teardown {
		delete(c.entries, key)
	}
	c.mu.Unlock()
	if teardown {
		entry.cancel()
	}
}

func (c *watchCache) get(ctxName string, gvr schema.GroupVersionResource, namespace, name string) (map[string]any, error) {
	if err := c.ctx.Err(); err != nil {
		return nil, err
	}
	informer, err := c.informerForGet(ctxName, gvr, namespace, name)
	if err != nil {
		return nil, err
	}
	key := name
	if namespace != "" {
		key = namespace + "/" + name
	}
	obj, exists, err := informer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%s %q not found", gvr.Resource, name)
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected object type %T", obj)
	}
	return u.Object, nil
}

func (c *watchCache) list(ctxName string, gvr schema.GroupVersionResource, namespace string) (map[string]any, error) {
	if err := c.ctx.Err(); err != nil {
		return nil, err
	}
	informer, err := c.informerFor(informerKey{context: ctxName, gvr: gvr, namespace: namespace})
	if err != nil {
		return nil, err
	}
	stored := informer.GetStore().List()
	items := make([]any, 0, len(stored))
	for _, obj := range stored {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		items = append(items, u.Object)
	}
	return map[string]any{
		"apiVersion": gvr.GroupVersion().String(),
		"kind":       "List",
		"items":      items,
	}, nil
}

func (c *watchCache) informerForGet(ctxName string, gvr schema.GroupVersionResource, namespace, name string) (cache.SharedIndexInformer, error) {
	listKey := informerKey{context: ctxName, gvr: gvr, namespace: namespace}
	c.mu.Lock()
	entry, ok := c.entries[listKey]
	c.mu.Unlock()
	if ok {
		return entry.informer, nil
	}
	return c.informerFor(informerKey{context: ctxName, gvr: gvr, namespace: namespace, name: name})
}

func (c *watchCache) informerFor(key informerKey) (cache.SharedIndexInformer, error) {
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		c.mu.Unlock()
		return entry.informer, nil
	}
	if pending, ok := c.pending[key]; ok {
		c.mu.Unlock()
		<-pending.done
		return pending.informer, pending.err
	}
	pending := &pendingEntry{done: make(chan struct{})}
	c.pending[key] = pending
	c.mu.Unlock()

	informer, cancel, err := c.startInformer(key)
	pending.informer, pending.err = informer, err
	close(pending.done)

	c.mu.Lock()
	delete(c.pending, key)
	if err == nil {
		c.entries[key] = &watchEntry{informer: informer, cancel: cancel}
	}
	c.mu.Unlock()

	return informer, err
}

func (c *watchCache) startInformer(key informerKey) (cache.SharedIndexInformer, context.CancelFunc, error) {
	cl, err := c.clients.get(key.context)
	if err != nil {
		return nil, nil, err
	}
	informer := newDynamicInformer(cl.dynamic, key)
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.emit(key, obj) },
		UpdateFunc: func(_, obj any) { c.emit(key, obj) },
		DeleteFunc: func(obj any) { c.emit(key, obj) },
	})
	if err != nil {
		return nil, nil, err
	}

	entryCtx, cancel := context.WithCancel(c.ctx)
	c.wg.Go(func() { informer.RunWithContext(entryCtx) })

	if !cache.WaitForNamedCacheSyncWithContext(entryCtx, informer.HasSynced) {
		cancel()
		return nil, nil, fmt.Errorf("failed to sync watch for %s %s (namespace %q)", key.context, key.gvr.String(), key.namespace)
	}

	return informer, cancel, nil
}

func newDynamicInformer(dyn dynamic.Interface, key informerKey) cache.SharedIndexInformer {
	resource := dyn.Resource(key.gvr)
	tweak := func(opts *metav1.ListOptions) {
		if key.name != "" {
			opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", key.name).String()
		}
	}
	var listWatch *cache.ListWatch
	if key.namespace != "" {
		namespaced := resource.Namespace(key.namespace)
		listWatch = &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				tweak(&opts)
				return namespaced.List(context.Background(), opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				tweak(&opts)
				return namespaced.Watch(context.Background(), opts)
			},
		}
	} else {
		listWatch = &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				tweak(&opts)
				return resource.List(context.Background(), opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				tweak(&opts)
				return resource.Watch(context.Background(), opts)
			},
		}
	}
	return cache.NewSharedIndexInformer(listWatch, &unstructured.Unstructured{}, 10*time.Minute, cache.Indexers{})
}

func (c *watchCache) emit(key informerKey, obj any) {
	if c.changes == nil {
		return
	}
	listKey := informerKey{context: key.context, gvr: key.gvr, namespace: key.namespace}
	keys := []jpoetwatch.InvocationKey{fromInformerKey(listKey)}
	if name := objectName(obj); name != "" {
		keys = append(keys, fromInformerKey(informerKey{context: key.context, gvr: key.gvr, namespace: key.namespace, name: name}))
	}
	c.changes(keys)
}

func objectName(obj any) string {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = d.Obj
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	return u.GetName()
}

func fromInformerKey(key informerKey) jpoetwatch.InvocationKey {
	return jpoetwatch.InvocationKey(fmt.Sprintf("kubernetes://%s/%s/%s/%s/%s/%s", key.context, key.gvr.Group, key.gvr.Version, key.gvr.Resource, key.namespace, key.name))
}

func toInformerKey(key jpoetwatch.InvocationKey) (informerKey, bool) {
	const prefix = "kubernetes://"
	s := string(key)
	if !strings.HasPrefix(s, prefix) {
		return informerKey{}, false
	}
	parts := strings.SplitN(s[len(prefix):], "/", 6)
	if len(parts) != 6 {
		return informerKey{}, false
	}
	gvr := schema.GroupVersionResource{Group: parts[1], Version: parts[2], Resource: parts[3]}
	return informerKey{context: parts[0], gvr: gvr, namespace: parts[4], name: parts[5]}, true
}
