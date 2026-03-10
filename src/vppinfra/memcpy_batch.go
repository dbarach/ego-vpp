package clib

import (
	"cpu"
	"ego"
	"sync/atomic"
	"unsafe"
)

// #include <vppinfra/memcpy_batch.h>
import "C"

// unaligned cache line
// +ego { align 1 }
type uline vec[[cpu.CacheLineBytes]byte]

// +ego always_inline
func (l *uline) ptr(i int) unsafe.Pointer { return unsafe.Add(unsafe.Pointer(l), i) }

// +ego always_inline
func (l *uline) line(i, n int) *uline { return (*uline)(l.ptr(i + n*cpu.CacheLineBytes)) }

const copy_small_max = 128 + 64

func (dst *uline) copy_small(src *uline, n_bytes int) {
	type (
		v32 vec[[32]uint8] // +ego { align 1 }
		m32 vec[[32]bool]
		v64 vec[[64]uint8] // +ego { align 1 }
		m64 vec[[64]bool]
	)
	mask := uint64(1)<<(n_bytes%64) - 1
	if n_bytes < 64 {
		if n_bytes <= 32 {
			ok := m32(uint32(mask))
			ifelse(ok, *(*v32)(dst.ptr(0))) = ifelse(ok, *(*v32)(src.ptr(0)))
		} else {
			ok := m64(mask)
			ifelse(ok, *(*v64)(dst.ptr(0))) = ifelse(ok, *(*v64)(src.ptr(0)))
		}
	} else {
		const n = 64
		ok := m64(mask)
		if n_bytes < 128 {
			*(*v64)(dst.ptr(0)) = *(*v64)(src.ptr(0))
			ifelse(ok, *(*v64)(dst.ptr(1 * n))) = ifelse(ok, *(*v64)(src.ptr(1 * n)))
		} else {
			*(*v64)(dst.ptr(0 * n)) = *(*v64)(src.ptr(0 * n))
			*(*v64)(dst.ptr(1 * n)) = *(*v64)(src.ptr(1 * n))
			ifelse(ok, *(*v64)(dst.ptr(2 * n))) = ifelse(ok, *(*v64)(src.ptr(2 * n)))
		}
	}
}

// +ego always_inline
func (dst *uline) copy(src *uline, n_bytes int) {
	if n_bytes < copy_small_max {
		dst.copy_small(src, n_bytes)
		return
	}

	const lb = cpu.CacheLineBytes
	i := 0
	n_left := n_bytes

	// Copy 8 cache lines from unaligned source to aligned destination.
	{
		const n = 8
		for n_left >= n*lb {
			var l [4]uline
			l[0], l[1], l[2], l[3] = *src.line(i, 0), *src.line(i, 1), *src.line(i, 2), *src.line(i, 3)
			*dst.line(i, 0) = l[0]
			*dst.line(i, 1) = l[1]
			*dst.line(i, 2) = l[2]
			*dst.line(i, 3) = l[3]
			l[0], l[1], l[2], l[3] = *src.line(i, 4), *src.line(i, 5), *src.line(i, 6), *src.line(i, 7)
			*dst.line(i, 4) = l[0]
			*dst.line(i, 5) = l[1]
			*dst.line(i, 6) = l[2]
			*dst.line(i, 7) = l[3]
			i += n * lb
			n_left -= n * lb
		}
	}
	{
		const n = 4
		for n_left >= n*lb {
			*dst.line(i, 0) = *src.line(i, 0)
			*dst.line(i, 1) = *src.line(i, 1)
			*dst.line(i, 2) = *src.line(i, 2)
			*dst.line(i, 3) = *src.line(i, 3)
			i += n * lb
			n_left -= n * lb
		}
	}

	{
		const n = 2
		for n_left >= n*lb {
			*dst.line(i, 0) = *src.line(i, 0)
			*dst.line(i, 1) = *src.line(i, 1)
			i += n * lb
			n_left -= n * lb
		}
	}

	{
		const n = 1
		for n_left >= n*lb {
			*dst.line(i, 0) = *src.line(i, 0)
			i += n * lb
			n_left -= n * lb
		}
	}

	if n_left > 0 {
		(*uline)(dst.ptr(i)).copy_small((*uline)(src.ptr(i)), n_left)
	}
}

// Go version of clib_memcpy_batch_single.
// +ego { alias(clib_memcpy_batch_single) }
func copy_batch_single(dst, src *uline, n_bytes int) { dst.copy(src, n_bytes) }

type (
	// Ordered as arguments to standard C memcpy.
	mem_copy_buffer struct {
		dst, src   unsafe.Pointer
		n_bytes    uint32
		aux_enable uint32
		aux_data   [5]uintptr
	}
	mem_copy_aux_op uint32
	mem_copy_stage  struct {
		src, dst           *uline
		n_copy_bytes_left  uint32
		n_bytes_prefetched uint32
	}
	mem_copy_config struct {
		enable_prefetch bool
		debug           bool

		prefetch_options cpu.PrefetchOptions

		prefetch_stride uint

		// Prefetch this many bytes at start of each buffer.
		n_prefetch_bytes_per_buffer uint

		// Copy/prefetch up to this many cache lines per stage.
		n_copy_bytes_per_stage uint

		batch *C.clib_memcpy_batch_t
	}
)

// +ego always_inline
func (s *mem_copy_stage) set(b *mem_copy_buffer) {
	s.src = (*uline)(b.src)
	s.dst = (*uline)(b.dst)
	s.n_copy_bytes_left = uint32(b.n_bytes)
	s.n_bytes_prefetched = 0
}

// +ego always_inline
func (s *mem_copy_stage) n_prefetch_bytes(cf mem_copy_config) int {
	n_left := int(s.n_copy_bytes_left)
	if nb := cf.n_prefetch_bytes_per_buffer; nb != ^uint(0) && n_left >= int(nb) {
		n_left = int(nb)
	}
	return n_left
}

// +ego always_inline
func (s *mem_copy_stage) prefetch_done(cf mem_copy_config) bool {
	return !(cf.enable_prefetch && int(s.n_bytes_prefetched) < s.n_prefetch_bytes(cf))
}

// +ego { always_inline }
func (cf mem_copy_config) copy(bs []mem_copy_buffer) {
	nb := uint32(len(bs))
	if nb == 0 {
		return
	}
	const lb = cpu.CacheLineBytes
	bytes_per_stage := int(cf.n_copy_bytes_per_stage)
	lines_per_stage := (bytes_per_stage + lb - 1) / lb
	var stages_ [32]mem_copy_stage
	stages := stages_[:min(1+cf.prefetch_stride, uint(len(stages_)))]
	// Prefetch, copy buffer indices.
	pbi, cbi := 0, 0
	// Prefetch, copy stage indices.
	psi, csi := 0, 0
	// Number of stages with either prefetch or copy active.
	n_active_stages := 1
	stages[psi].set(&bs[pbi])
	pbi++
	for iter := 0; ; iter++ {
		// Prefetch first chunk(s) of buffer.  Hardware prefetcher
		// will eventually take over so we don't need to prefetch
		// everything for large buffers.
		if cf.enable_prefetch {
			ps := &stages[psi]
			nb := int(ps.n_bytes_prefetched)
			n_bytes_left := ps.n_prefetch_bytes(cf)
			if nb < n_bytes_left {
				sp, dp := ps.src.ptr(nb), ps.dst.ptr(nb)
				n_pref := min(n_bytes_left-nb, bytes_per_stage)
				for i := 0; i < n_pref; i += lb {
					cpu.PrefetchPtrWithOptions(unsafe.Add(sp, i), cf.prefetch_options)
					cpu.PrefetchPtrWithOptions(unsafe.Add(dp, i), cf.prefetch_options)
				}
				nb += n_pref
				ps.n_bytes_prefetched = uint32(nb)
			}
			if nb >= n_bytes_left && n_active_stages < len(stages) && pbi < len(bs) {
				psi++
				if psi >= len(stages) {
					psi = 0
				}
				stages[psi].set(&bs[pbi])
				pbi++
				n_active_stages++
			}
		}

		// Copy one chunk worth of cache lines (for a previously prefetched buffer).
		cs := &stages[csi]
		if iter >= len(stages)-1 && cs.prefetch_done(cf) {
			nb := int(cs.n_copy_bytes_left)
			n_this_stage := min(bytes_per_stage, nb)
			is_last_copy_this_buffer := n_this_stage == nb
			cs.n_copy_bytes_left = uint32(nb - n_this_stage)
			cs.dst.copy(cs.src, n_this_stage)
			if !is_last_copy_this_buffer {
				// Advance src and dst for next copy stage.
				cs.src = cs.src.line(0, lines_per_stage)
				cs.dst = cs.dst.line(0, lines_per_stage)
			} else {
				bs[cbi].do_aux(cf)
				cbi++
				// Done copying final buffer?
				if cbi >= len(bs) {
					break
				}
				csi++
				if csi >= len(stages) {
					csi = 0
				}
				n_active_stages--
				if !cf.enable_prefetch {
					stages[csi].set(&bs[cbi])
					n_active_stages++
				}
			}
		}
	}
}

// Perform aux operation after buffer copy is done.
// +ego always_inline
func (a *mem_copy_buffer) do_aux(cf mem_copy_config) {
	if a.aux_enable == 0 {
		return
	}
	switch C.int(cf.batch.aux_op) {
	case C.clib_memcpy_batch_aux_op_atomic_store_uint32_release:
		atomic.StoreWithOptions((*uint32)(unsafe.Pointer(a.aux_data[0])), uint32(a.aux_data[1]),
			atomic.Options{Ordering: atomic.Release},
		)
	case C.clib_memcpy_batch_aux_op_atomic_store_uint64_release:
		atomic.StoreWithOptions((*uint64)(unsafe.Pointer(a.aux_data[0])), uint64(a.aux_data[1]),
			atomic.Options{Ordering: atomic.Release},
		)
	case C.clib_memcpy_batch_aux_op_call_aux_func:
		// Call user's function from C code.
		c := ego.ClosureStruct{Func: unsafe.Pointer(cf.batch.aux_func)}
		(*(*func(*C.clib_memcpy_batch_t, unsafe.Pointer))(unsafe.Pointer(&c)))(cf.batch, unsafe.Pointer(a)) // +ego call-no-opaque
	}
}

// Go version of clib_memcpy_batch_sync.
// +ego { alias(clib_memcpy_batch_sync) }
func copy_batch_sync(b *C.clib_memcpy_batch_t) {
	if b != nil && b.n_calls > 0 {
		bs := unsafe.Slice((*mem_copy_buffer)(unsafe.Pointer(&b.calls[0])), b.n_calls)
		if b.prefetch_memcpy_likely_profitable != 0 {
			cf := mem_copy_config{
				enable_prefetch: true,
				// Not worth prefetch into l1 (assume distance small between l1 and l2)
				// Seems true for x86_64 not sure about others.  Probably machine dependent.
				prefetch_options:            cpu.PrefetchOptions{Locality: cpu.PrefetchL2},
				prefetch_stride:             4,
				n_prefetch_bytes_per_buffer: ^uint(0),
				n_copy_bytes_per_stage:      1 << 10,
				batch:                       b,
			}
			cf.copy(bs)
		} else {
			cf := mem_copy_config{
				enable_prefetch:        false,
				n_copy_bytes_per_stage: 1 << 31, // copy everything with no prefetch
				batch:                  b,
			}
			cf.copy(bs)
		}
		b.n_calls = 0
	}
}
