/* SPDX-License-Identifier: MIT
 * Copyright (c) 2026 Eliot Dresselhaus
 */

#ifndef included_spsc_fifo_h
#define included_spsc_fifo_h

#include <vppinfra/types.h>

/* Single producer single consumer fifos. */

/* Head/tail pointer for fifo. */
typedef struct {
  uword v;
} __attribute__ ((aligned (CLIB_CACHE_LINE_BYTES))) spsc_fifo_ptr_t;

#define SPSC_FIFO_DECL(T, L)			\
  struct {					\
    /* Head/tail pointers (cache aligned). */	\
    spsc_fifo_ptr_t p[2];			\
    /* 2^L fixed elements of data follow. */	\
    T d[1 << (L)];				\
  }

#define SPSC_FIFO_PUSH(F, V)					\
({								\
  uword __tail = clib_atomic_load_relax_n (&(F)->p[1].v);	\
  uword __head = clib_atomic_load_acq_n (&(F)->p[0].v);		\
  uword __ok = __tail - __head < ARRAY_LEN ((F)->d);		\
  if (__ok)							\
    {								\
      f->d[__tail % ARRAY_LEN ((F)->d)] = *(V);			\
      clib_atomic_store_rel_n (&(F)->p[1].v, __tail + 1);	\
    }								\
  __ok;								\
})

#define SPSC_FIFO_POP(F, V)					\
({								\
  uword __head = clib_atomic_load_relax_n (&(F)->p[0].v);	\
  uword __tail = clib_atomic_load_acq_n (&(F)->p[1].v);		\
  uword __ok = __tail != __head;				\
  if (__ok)							\
    {								\
      *(V) = f->d[__head % ARRAY_LEN ((F)->d)];			\
      clib_atomic_store_rel_n (&(F)->p[0].v, __head + 1);	\
    }								\
  __ok;								\
})

#endif /* included_spsc_fifo_h */
