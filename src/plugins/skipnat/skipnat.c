/*
 * skipnat.c - skeleton vpp engine plug-in
 *
 * Copyright (c) <current-year> <your-organization>
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

#include <vnet/vnet.h>
#include <vnet/plugin/plugin.h>
#include <skipnat/skipnat.h>

#include <vlibapi/api.h>
#include <vlibmemory/api.h>
#include <vpp/app/version.h>
#include <stdbool.h>

skipnat_main_t skipnat_main;

static clib_error_t * skipnat_init (vlib_main_t * vm)
{
  skipnat_main_t * smp = &skipnat_main;
  clib_error_t * error = 0;

  smp->vlib_main = vm;
  smp->vnet_main = vnet_get_main();

  return error;
}

VLIB_INIT_FUNCTION (skipnat_init);

/* *INDENT-OFF* */
VNET_FEATURE_INIT (skipnat, static) =
{
  .arc_name = "ip4-unicast",
  .node_name = "skipnat",
  /* $$$$ might need VNET_FEATURE_ARC_ORDER, but not in hgw case */
  .runs_before = VNET_FEATURES ("cnat-snat-ip4"),
};
/* *INDENT-ON */

/* *INDENT-OFF* */
VLIB_PLUGIN_REGISTER () =
{
  .version = VPP_BUILD_VER,
  .description = "skip nat plugin for 192.168.0.0/16"
};
/* *INDENT-ON* */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
