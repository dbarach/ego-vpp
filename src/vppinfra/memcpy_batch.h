/*
 * Copyright (c) 2025 Cisco and/or its affiliates.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
/*
  Copyright (c) 2025 Eliot Dresselhaus

  Permission is hereby granted, free of charge, to any person obtaining
  a copy of this software and associated documentation files (the
  "Software"), to deal in the Software without restriction, including
  without limitation the rights to use, copy, modify, merge, publish,
  distribute, sublicense, and/or sell copies of the Software, and to
  permit persons to whom the Software is furnished to do so, subject to
  the following conditions:

  The above copyright notice and this permission notice shall be
  included in all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
  EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
  MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
  NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
  LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
  OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
  WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

#ifndef included_clib_memcpy_batch_h
#define included_clib_memcpy_batch_h

#include <vppinfra/clib.h>
#include <vppinfra/string.h>

#define foreach_clib_memcpy_batch_aux_op	\
  _ (nothing)					\
  _ (call_aux_func)				\
  _ (atomic_store_uint32_release)		\
  _ (atomic_store_uint64_release)

typedef enum {
#define _(f) clib_memcpy_batch_aux_op_##f,
  foreach_clib_memcpy_batch_aux_op
#undef _
} clib_memcpy_batch_aux_op_t;

/* Arguments to memcpy as a struct. */
typedef struct {
  void *restrict dst;
  const void *restrict src;
  u32 n;
  u32 aux_enable;
  /* Aux operation to perform after this memcpy is done. */
  uword aux_data[5];
} clib_memcpy_batch_call_t;

struct clib_memcpy_batch_s;
typedef void (clib_memcpy_batch_aux_func_t) (struct clib_memcpy_batch_s *b, clib_memcpy_batch_call_t *c);

#define clib_memcpy_batch_n_fixed_calls 1 /* 0 breaks ego cgo interface */

typedef struct clib_memcpy_batch_s {
  uword n_calls;
  uword n_calls_alloc;
  uword prefetch_memcpy_likely_profitable; /* e.g. you have memory bandwidth pressure (lots of sessions etc) */
  clib_memcpy_batch_aux_func_t * aux_func;
  clib_memcpy_batch_aux_op_t aux_op;
  clib_memcpy_batch_call_t calls[clib_memcpy_batch_n_fixed_calls]; /* actually larger */
} clib_memcpy_batch_t;

/* Override with prefetching version written in Go. */
void clib_memcpy_batch_sync (clib_memcpy_batch_t *b);
void clib_memcpy_batch_single (void *restrict dst, const void *restrict src, size_t n);

#define CLIB_MEMCPY_NO_BATCH ((clib_memcpy_batch_t *) 0)

/* Allocate batch either on heap or stack. */
#define _clib_memcpy_batch_alloc_bytes(n)				\
  (sizeof(clib_memcpy_batch_t) +					\
   ((n) - clib_memcpy_batch_n_fixed_calls)*sizeof(clib_memcpy_batch_call_t))
#define CLIB_MEMCPY_BATCH_ALLOC(n_alloc)				\
  clib_memcpy_batch_init(clib_mem_alloc_no_fail(_clib_memcpy_batch_alloc_bytes(n_alloc)), (n_alloc))
#define CLIB_MEMCPY_BATCH_ALLOC_STACK(n_alloc)				\
  clib_memcpy_batch_init(clib_mem_alloc_stack(_clib_memcpy_batch_alloc_bytes(n_alloc)), (n_alloc))

extern __thread u8 memcpy_batch_prefetch_likely_profitable;

static_always_inline clib_memcpy_batch_t *
clib_memcpy_batch_init (clib_memcpy_batch_t *b, uword n_alloc)
{
  if (b)
    {
      b->n_calls = 0;
      b->n_calls_alloc = n_alloc;
      b->aux_op = clib_memcpy_batch_aux_op_nothing;
      b->prefetch_memcpy_likely_profitable = memcpy_batch_prefetch_likely_profitable;
    }
  return b;
}

static_always_inline clib_memcpy_batch_call_t *
clib_memcpy_batch (clib_memcpy_batch_t *b, void *restrict dst, const void *restrict src, size_t n)
{
  clib_memcpy_batch_call_t * c = 0;
  if (!b || b->n_calls_alloc < 2)
    {
      clib_memcpy_batch_single (dst, src, n);
    }
  else
    {
      c = &b->calls[b->n_calls];
      c->dst = dst;
      c->src = src;
      c->n = (u32)n;
      c->aux_enable = 0;
      b->n_calls++;
      if (b->n_calls >= b->n_calls_alloc)
	clib_memcpy_batch_sync (b);
    }
  return c;
}

typedef struct {
  void * dst;
  uword value;
} clib_memcpy_batch_aux_atomic_store_t;

static_always_inline void
clib_memcpy_batch_atomic_store_release_u32 (clib_memcpy_batch_t *b, u32 *dst, u32 value)
{
  if (!b || b->n_calls == 0 || b->n_calls_alloc < 2)    /* n calls 0 can happen via previous sync */
    {
      clib_atomic_store_rel_n (dst, value);
    }
  else
    {
      clib_memcpy_batch_call_t *c = &b->calls[b->n_calls-1];
      clib_memcpy_batch_aux_atomic_store_t *a = (void *)(&c->aux_data);
      c->aux_enable = 1;
      a->dst = dst;
      a->value = value;
    }
}

static_always_inline void
clib_memcpy_batch_atomic_store_release_u64 (clib_memcpy_batch_t *b, u64 *dst, u64 value)
{
  if (!b || b->n_calls == 0 || b->n_calls_alloc < 2)    /* n calls 0 can happen via previous sync */
    {
      clib_atomic_store_rel_n (dst, value);
    }
  else
    {
      clib_memcpy_batch_call_t *c = &b->calls[b->n_calls-1];
      clib_memcpy_batch_aux_atomic_store_t *a = (void *)(&c->aux_data);
      c->aux_enable = 1;
      a->dst = dst;
      a->value = value;
    }
}

#endif /* included_clib_memcpy_batch_h */
