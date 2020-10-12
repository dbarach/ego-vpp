
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

option(VPP_BUILD_G2_PLUGIN "Build golang g2 plugin" ON)
option(VPP_USE_DEBUG_GTK "Use locally-installed gtk libs" OFF)
if(VPP_USE_DEBUG_GTK)
  list(APPEND GTK_LIBS /usr/local/lib/libgtk-3.so)
  list(APPEND GTK_LIBS /usr/local/lib/libgdk-3.so)
  list(APPEND GTK_LIBS /usr/local/lib/libcairo-gobject.so)
  list(APPEND GTK_LIBS /usr/local/lib/libcairo.so)
else()
  list(APPEND GTK_LIBS gtk-3)
  list(APPEND GTK_LIBS gdk-3)
  list(APPEND GTK_LIBS cairo-gobject)
  list(APPEND GTK_LIBS cairo)
endif()

if(VPP_BUILD_G2_PLUGIN)
  add_vpp_plugin(g2
    SOURCES
    g2.c
    ego_g2.o
    g2.h

    INSTALL_HEADERS
    g2.h

    LINK_LIBRARIES
    ${GTK_LIBS}
    pangocairo-1.0
    pango-1.0
    harfbuzz
    atk-1.0
    gdk_pixbuf-2.0
    gio-2.0
    gobject-2.0
    glib-2.0
    pthread
    )

  set(EGO_OPTIMIZE_FLAG "-O2")

  add_custom_command(
    OUTPUT ego_g2.o
    COMMAND ego -tags vpp -link-bitcode -build-dir `pwd` -g ${EGO_OPTIMIZE_FLAG} -fpic -o `pwd`/ego_g2.bc ego/tools/g2
    COMMAND clang -Werror -Wall -march=corei7 -mtune=corei7-avx -c -g ${EGO_OPTIMIZE_FLAG} -fstack-protector -DFORTIFY_SOURCE=2 -maes -mavx2 -fPIC `pwd`/ego_g2.bc
    COMMENT "build g2 golang code..."
    )

  add_custom_target(g2_rebuild
    COMMAND rm -f ego_g2.o
    )
endif()
