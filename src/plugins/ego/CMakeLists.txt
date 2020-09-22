
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

add_vpp_plugin(ego
  SOURCES
  ego.c
  ego.h
  ego_periodic.c
  ego_vpp.o
  node.c

  MULTIARCH_SOURCES
  node.c

  INSTALL_HEADERS
  ego.h

  API_FILES
  ego.api

  API_TEST_SOURCES
  ego_test.c
)

add_custom_command(
  OUTPUT ego_vpp.o
  COMMAND $ENV{GOPATH}/bin/ego -link-bitcode -build-dir `pwd` -g -O0 -o ego_vpp.bc -main ego/vpp.main ego/vpp
  COMMAND clang -Werror -Wall -march=corei7 -mtune=corei7-avx -c -g -O0 -fstack-protector -DFORTIFY_SOURCE=2 -maes -mavx2 -fPIC ego_vpp.bc
  COMMENT "build golang objects.."
)

add_custom_target(ego_rebuild
  COMMAND rm -f ego_vpp.o
  )
