
# Copyright (c)
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

option(VPP_BUILD_EGO_PLUGIN "Build the ego test plugin" OFF)
if(VPP_BUILD_EGO_PLUGIN)
  add_vpp_plugin(ego
    SOURCES
    ego.c
    ego.h
    ego_periodic.c
    ego_vpp.o
    node.c

    INSTALL_HEADERS
    ego.h

    API_FILES
    ego.api

    API_TEST_SOURCES
    ego_test.c

    COMPONENT
    vpp-plugin-ego
    )

  set(EGO_OPTIMIZE_FLAG "-O0")

  add_custom_command(
    OUTPUT ego_vpp.o
    COMMAND ego -link-bitcode -build-dir `pwd` -g ${EGO_OPTIMIZE_FLAG} -fpic -o `pwd`/ego_vpp.bc ego/test/vpp1
    COMMAND clang -Werror -Wall -march=corei7 -mtune=corei7-avx -c -g ${EGO_OPTIMIZE_FLAG} -fstack-protector -DFORTIFY_SOURCE=2 -maes -mavx2 -fPIC `pwd`/ego_vpp.bc
    COMMENT "build golang objects..."
    )

  add_custom_target(ego_rebuild
    COMMAND rm -f ego_vpp.o
    )
endif()
