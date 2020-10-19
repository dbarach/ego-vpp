
# Copyright (c) <current-year> <your-organization>
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

option(VPP_BUILD_EHS_PLUGIN "Build golang host-stack test plugin" OFF)
if(VPP_BUILD_EHS_PLUGIN)
  add_vpp_plugin(ehs
    SOURCES
    ego_ehs.o
    ehs.c

    COMPONENT
    vpp-plugin-ehs
    )

  set(EGO_OPTIMIZE_FLAG "-O0")

  add_custom_command(
    OUTPUT ego_ehs.o
    COMMAND ego -tags vpp -link-bitcode -build-dir `pwd` -g ${EGO_OPTIMIZE_FLAG} -fpic -o `pwd`/ego_ehs.bc ego/test/vhs1
    COMMAND clang -Werror -Wall -march=corei7 -mtune=corei7-avx -c -g ${EGO_OPTIMIZE_FLAG} -fstack-protector -DFORTIFY_SOURCE=2 -maes -mavx2 -fPIC `pwd`/ego_ehs.bc
    COMMENT "build golang host stack test code..."
    )

  add_custom_target(ehs_rebuild
    COMMAND rm -f ego_ehs.o
    )
endif()
