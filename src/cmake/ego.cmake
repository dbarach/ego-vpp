find_program(VPP_HAVE_EGO ego)
if (VPP_HAVE_EGO)
  execute_process(COMMAND ego print-goroot OUTPUT_VARIABLE ego_go_root)
  string(STRIP "${ego_go_root}" ego_go_root)
  include(${ego_go_root}/ego.cmake)

  # So C code can tell if ego is present or not.
  add_definitions(-DHAVE_EGO=1)

  # ego flags for build type.
  set(ego_flags_debug -g -O0)
  set(ego_flags_release -g -O2)
  set(ego_flags "${ego_flags_${CMAKE_BUILD_TYPE}}")
endif()
