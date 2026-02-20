/* SPDX-License-Identifier: MIT
 * Copyright (c) 2026 Eliot Dresselhaus
 */

#ifndef included_smp_fifo_h
#define included_smp_fifo_h

#include <vppinfra/atomics.h>

/* {Single,multiple} producer single consumer fifos. */

/* Head/tail pointer for fifo. */
typedef struct {
  uword v;
} __attribute__ ((aligned (CLIB_CACHE_LINE_BYTES))) clib_spsc_fifo_ptr_t;

#define CLIB_SPSC_FIFO_DECL(T, L)		\
  struct {					\
    /* Head/tail pointers (cache aligned). */	\
    clib_spsc_fifo_ptr_t p[2];			\
    /* 2^L fixed elements of data follow. */	\
    T d[1 << (L)];				\
  }

/* Add value V to fifo F.  Return 0 for full fifo; otherwise 1. */
#define CLIB_SPSC_FIFO_PUSH(F, V, L)				\
({								\
  uword __tail = clib_atomic_load_relax_n (&(F)->p[1].v);	\
  uword __head = clib_atomic_load_acq_n (&(F)->p[0].v);		\
  uword __len = __tail - __head;				\
  uword __ok = __len < ARRAY_LEN ((F)->d);			\
  if (__ok)							\
    {								\
      f->d[__tail % ARRAY_LEN ((F)->d)] = *(V);			\
      clib_atomic_store_rel_n (&(F)->p[1].v, __tail + 1);	\
    }								\
  if (L)							\
    {								\
      *(L) = __len;						\
    }								\
  __ok;								\
})

/* Remove value V to fifo F.  Return 0 for empty fifo; otherwise 1. */
#define CLIB_SPSC_FIFO_POP(F, V, L)				\
({								\
  uword __head = clib_atomic_load_relax_n (&(F)->p[0].v);	\
  uword __tail = clib_atomic_load_acq_n (&(F)->p[1].v);		\
  uword __len = __tail - __head;				\
  uword __ok = __len > 0;					\
  if (__ok)							\
    {								\
      *(V) = f->d[__head % ARRAY_LEN ((F)->d)];			\
      clib_atomic_store_rel_n (&(F)->p[0].v, __head + 1);	\
    }								\
  if (L)							\
    {								\
      *(L) = __len;						\
    }								\
  __ok;								\
})

#define CLIB_SPSC_FIFO_LEN(F)					\
({								\
  uword __head = clib_atomic_load_relax_n (&(F)->p[0].v);	\
  uword __tail = clib_atomic_load_relax_n (&(F)->p[1].v);	\
  __tail - __head;						\
})

#define CLIB_MPSC_FIFO_DECL(T, L)				\
  struct {							\
    /* Head/tail pointers (cache aligned). */			\
    clib_mpsc_fifo_header64_t h64;				\
    /* 2^L fixed elements of cache-aligned data follow. */	\
    struct {							\
      CLIB_CACHE_LINE_ALIGN_MARK (cacheline0);			\
      T __t;							\
    } d[1 << (L)];						\
  }

/* Head/tail counter plus lock bit for fifo.
   Layout as follows:
     head [63:33] 31 bits
     unused [32]
     tail [31:1]  31 bits
     lock [0] 1 bit */
typedef struct {
  u64 v;
} __attribute__ ((aligned (CLIB_CACHE_LINE_BYTES))) clib_mpsc_fifo_header64_t;

always_inline void
clib_mpsc_fifo_get_head_tail (u64 v, u64 *head, u64 *tail)
{
  *head = (v >> (1*32 + 1)) & 0x7fffffff;
  *tail = (v >> (0*32 + 1)) & 0x7fffffff;
}

always_inline void
clib_mpsc_fifo_lock (clib_mpsc_fifo_header64_t *h64, u64 *head, u64 *tail)
{
  u64 v = clib_atomic_load_relax_n (&h64->v);

  while (1)
    {
      /* Someone else has lock => wait a bit and try again? */
      if (v & 1)
	{
	  CLIB_PAUSE ();
	  v = clib_atomic_load_relax_n (&h64->v);
	}

      /* Otherwise set lock bit and preserve head and tail pointers. */
      else if (clib_atomic_cmp_and_swap_acq_relax_n (&h64->v, &v, v | 1, /* weak */ 0))
	break;
    }

  clib_mpsc_fifo_get_head_tail (v, head, tail);
}

always_inline void
clib_mpsc_fifo_unlock (clib_mpsc_fifo_header64_t *h64, u64 head, u64 tail)
{
  u64 v = (head << (1*32 + 1)) + (tail << (0*32 + 1));
  clib_atomic_store_rel_n (&h64->v, v);
}

/* Might be locked or not.  But len should still be valid. */
#define CLIB_MPSC_FIFO_LEN(F)					\
({								\
  u64 __head, __tail;						\
  clib_mpsc_fifo_get_head_tail ((F)->h64.v, &__head, &__tail);	\
  __tail - __head;						\
})

/* Add value V to fifo F.  Return 0 for full fifo; otherwise 1. */
#define CLIB_MPSC_FIFO_PUSH(F, V, L)			\
({							\
  u64 __head, __tail;					\
  uword __len, __ok;					\
  clib_mpsc_fifo_lock (&(F)->h64, &__head, &__tail);	\
  __len = __tail - __head;				\
  __ok = __len < ARRAY_LEN ((F)->d);			\
  if (__ok)						\
    {							\
      f->d[__tail % ARRAY_LEN ((F)->d)].__t = *(V);	\
      __tail++;						\
    }							\
  clib_mpsc_fifo_unlock (&(F)->h64, __head, __tail);	\
  if (L)						\
    {							\
      *(L) = __len;					\
    }							\
  __ok;							\
})

#define CLIB_MPSC_FIFO_POP(F, V, L)			\
({							\
  u64 __head, __tail;					\
  uword __len, __ok;					\
  clib_mpsc_fifo_lock (&(F)->h64, &__head, &__tail);	\
  __len = __tail - __head;				\
  __ok = __len > 0;					\
  if (__ok)						\
    {							\
      *(V) = f->d[__head % ARRAY_LEN ((F)->d)].__t;	\
      __head++;						\
    }							\
  clib_mpsc_fifo_unlock (&(F)->h64, __head, __tail);	\
  if (L)						\
    {							\
      *(L) = __len;					\
    }							\
  __ok;							\
})

#endif /* included_smp_fifo_h */
