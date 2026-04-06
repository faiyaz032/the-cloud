package hashing

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
)

type HashRing struct {
	Nodes    []uint32
	NodeMap  map[uint32]string
	Replicas int
}

func NewHashRing(replicas int) *HashRing {
	return &HashRing{
		NodeMap:  make(map[uint32]string),
		Replicas: replicas,
	}
}

func (r *HashRing) hashFn(key string) uint32 {
	h := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(h[:4])
}

func (r *HashRing) AddNode(nodeIP string) {
	for i := 0; i < r.Replicas; i++ {
		vNodeName := fmt.Sprintf("%s#%d", nodeIP, i)
		hash := r.hashFn(vNodeName)
		r.Nodes = append(r.Nodes, hash)
		r.NodeMap[hash] = nodeIP
	}
	sort.Slice(r.Nodes, func(i, j int) bool {
		return r.Nodes[i] < r.Nodes[j]
	})
}

func (r *HashRing) GetNode(key string) string {
	if len(r.Nodes) == 0 {
		return ""
	}
	hash := r.hashFn(key)
	idx := sort.Search(len(r.Nodes), func(i int) bool {
		return r.Nodes[i] >= hash
	})
	if idx == len(r.Nodes) {
		idx = 0
	}
	return r.NodeMap[r.Nodes[idx]]
}
