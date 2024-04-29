
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

option(VPP_BUILD_G2F_PLUGIN "Build golang g2f plugin" ON)
option(VPP_USE_DEBUG_FLTK "Use locally-installed fltk libs" ON)
if(VPP_USE_DEBUG_FLTK)
  list(APPEND FLTK_LIBS /usr/local/lib/libfltk.so)
  list(APPEND FLTK_LIBS /usr/local/lib/libfltk_forms.so)
  list(APPEND FLTK_LIBS /usr/local/lib/libfltk_cairo.so)
  list(APPEND FLTK_LIBS /usr/local/lib/libfltk_images.so)
else()
endif()

if(VPP_BUILD_G2F_PLUGIN)
  include(../../cmake/ego.cmake)

  add_vpp_plugin(g2f
    SOURCES
    g2.c
    ego_g2f.o
    g2.h

    INSTALL_HEADERS
    g2.h

    LINK_LIBRARIES
    ${FLTK_LIBS}
    pangocairo-1.0
    pango-1.0
    harfbuzz
    atk-1.0
    cairo
    pthread

    COMPONENT
    vpp-plugin-g2f
    )

  ego_plugin_object(ego_g2f
    PACKAGE tools/g2f
    FLAGS -g;-O2;
    )

endif()
