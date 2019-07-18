/*
 * Copyright (c) 2019 Cisco and/or its affiliates.
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

#include <vppinfra/error.h>
#include <vppinfra/unix.h>
#include <vppinfra/format.h>

typedef struct
{
  int verbose;
  u8 *filename;
  u8 *contents;
} chewmap_main_t;

chewmap_main_t chewmap_main;

clib_error_t *chewmap (unformat_input_t *i, chewmap_main_t *cm)
{
  clib_error_t *error;
  unformat_input_t _f, *f = &_f;
  unformat_input_t _l, *l = &_l;
  u64 low_va, high_va, total_vspace = 0ULL;
  int va_range_set =0;
  u8 *cur;

  while (unformat_check_input (i) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat (i, "in %s", &cm->filename))
        ;
      else if (unformat (i, "verbose %=", &cm->verbose, 1))
        ;
      else
        return clib_error_return (0, "usage: chewmap in <fn> [verbose [<nn]]");
    }

  if (cm->filename == 0)
    return clib_error_return (0, "'in <filename>' missing, required");

  if ((error = clib_file_contents ((char *)(cm->filename), &cm->contents)))
    return error;

  unformat_init_vector (f, cm->contents);

  while (unformat_check_input (f) != UNFORMAT_END_OF_INPUT)
    {
      if (!unformat_user (f, unformat_line_input, l))
        continue;

      vec_add1 (l->buffer, 0);

      va_range_set = 0;

      if (unformat (l, "%llx-%llx", &low_va, &high_va))
        {
          va_range_set = 1;
          total_vspace += high_va - low_va;
        }

      if (va_range_set)
        {
          cur = l->buffer + l->index;

          while (cur < l->buffer + vec_len(l->buffer) - 1)
            {
              if (*cur == '/' || *cur == '[')
                {
                  fformat (stdout, "%016llx - %016llx [%8U] %s\n",
                           low_va, high_va, format_memory_size,
                           high_va - low_va, cur);
                  goto printed;
                }
              cur++;
            }
          fformat (stdout, "%016llx - %016llx [%8U]\n",
                   low_va, high_va, format_memory_size, high_va - low_va);
        }
    printed:
      unformat_free (l);
    }

  unformat_free (f);
  vec_free (cm->filename);

  fformat (stdout, "Total virtual space: %U\n", format_memory_size,
           total_vspace);
  return 0;
}


int main (int argc, char **argv)
{
  unformat_input_t i;
  clib_error_t *error;
  chewmap_main_t *cm = &chewmap_main;

  clib_mem_init (0, 128ULL << 20);

  unformat_init_command_line (&i, argv);
  error = chewmap (&i, cm);
  unformat_free (&i);

  if (error)
    {
      clib_error_report (error);
      return 1;
    }
  return 0;
}

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
