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

#include <vppinfra/mem.h>
#include <vppinfra/string.h>

/* Thread local since each thread can have different ideas of memory pressure. */
__clib_export __thread u8 memcpy_batch_prefetch_likely_profitable = 0;

#if !defined(HAVE_EGO) || HAVE_EGO == 0

__clib_export void clib_memcpy_batch_single (void *restrict dst, const void *restrict src, size_t n)
{ clib_memcpy_fast (dst, src, n); }

/* Fallback C implementation with no prefetching between calls.
   Meant to be overridden with another (e.g. Ego) implementation
   which does some smarter prefetching between calls. */
__clib_export void clib_memcpy_batch_sync (clib_memcpy_batch_t *b)
{
    if (!b)			/* disabled */
        return;
    for (uword i = 0; i < b->n_calls; i++)
      {
        clib_memcpy_batch_call_t *c = &b->calls[i];
        clib_memcpy_fast (c->dst, c->src, c->n);
        switch (b->aux_op)
	  {
	  case clib_memcpy_batch_aux_op_nothing:
            /* nothing to do */;
	    break;
	  case clib_memcpy_batch_aux_op_atomic_store_uint32_release:
	    if (c->aux_enable != 0)
	      clib_atomic_store_rel_n (uword_to_pointer(c->aux_data[0], u32 *), (u32) c->aux_data[1]);
	    break;
	  case clib_memcpy_batch_aux_op_atomic_store_uint64_release:
	    if (c->aux_enable != 0)
	      clib_atomic_store_rel_n (uword_to_pointer(c->aux_data[0], u64 *), (u64) c->aux_data[1]);
	    break;
	  case clib_memcpy_batch_aux_op_call_aux_func:
	    b->aux_func (b, c);
	    break;
	  }
      }
    b->n_calls = 0;
}

#endif /* !HAVE_EGO */
