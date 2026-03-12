find_program(VPP_HAVE_EGO egoNOT)
if (VPP_HAVE_EGO)
  execute_process(COMMAND ego print-goroot OUTPUT_VARIABLE ego_go_root)
  string(STRIP "${ego_go_root}" ego_go_root)
  include(${ego_go_root}/ego.cmake)

  # So C code can tell if ego is present or not.
  add_definitions(-DHAVE_EGO=1)

  # ego flags for build type.
  set(ego_flags_debug -g -O0 -cflags -DCLIB_DEBUG=1)
  set(ego_flags_release -g -O3)
  set(ego_flags "${ego_flags_${CMAKE_BUILD_TYPE}}")

  set(vpp_ego_flags_debug -g -O0 -cflags -DCLIB_DEBUG=1)
  set(vpp_ego_flags_release -g -O3 -lto)
  set(vpp_ego_flags "${vpp_ego_flags_${CMAKE_BUILD_TYPE}}")

  macro(add_ego_vpp_plugin name)
    cmake_parse_arguments(PLUGIN
      ""
      "COMPONENT;DEV_COMPONENT"
      "SOURCES;SUPPORTED_OS_LIST"
      ${ARGN}
    )
    set(_vpp_exclude_plugin 0)
    if(DEFINED VPP_EXCLUDED_PLUGINS AND NOT VPP_EXCLUDED_PLUGINS STREQUAL "")
      get_property(_vpp_plugins_exclude GLOBAL PROPERTY VPP_PLUGINS_EXCLUDE)
      list(FIND _vpp_plugins_exclude ${name} _vpp_exclude_idx)
      if(NOT _vpp_exclude_idx EQUAL -1)
	list(REMOVE_AT _vpp_plugins_exclude ${_vpp_exclude_idx})
	set_property(GLOBAL PROPERTY VPP_PLUGINS_EXCLUDE "${_vpp_plugins_exclude}")
	set(_vpp_exclude_plugin 1)
      endif()
    endif()
    set_property(GLOBAL APPEND PROPERTY VPP_PLUGINS_ALL_LIST ${name})
    if(DEFINED VPP_PLUGINS AND NOT VPP_PLUGINS STREQUAL "")
      if(VPP_PLUGINS STREQUAL "none")
	return()
      endif()
      get_property(_vpp_plugins_filter GLOBAL PROPERTY VPP_PLUGINS_FILTER)
      list(FIND _vpp_plugins_filter ${name} _vpp_plugin_idx)
      if(_vpp_plugin_idx EQUAL -1)
	return()
      endif()
      list(REMOVE_AT _vpp_plugins_filter ${_vpp_plugin_idx})
      set_property(GLOBAL PROPERTY VPP_PLUGINS_FILTER "${_vpp_plugins_filter}")
    elseif(DEFINED VPP_EXCLUDED_PLUGINS AND NOT VPP_EXCLUDED_PLUGINS STREQUAL "")
      if(_vpp_exclude_plugin)
	return()
      endif()
    endif()
    if (PLUGIN_SUPPORTED_OS_LIST AND NOT ${CMAKE_SYSTEM_NAME} IN_LIST PLUGIN_SUPPORTED_OS_LIST)
      message(WARNING "unsupported OS - ${name} plugin disabled")
      return()
    endif()
    set(plugin_name ${name}_plugin)
    set(api_includes)
    if(NOT PLUGIN_COMPONENT)
      set(PLUGIN_COMPONENT vpp-plugin-core)
    endif()
    if(NOT PLUGIN_DEV_COMPONENT)
      if(NOT VPP_EXTERNAL_PROJECT)
	set(PLUGIN_DEV_COMPONENT vpp-dev)
      else()
	set(PLUGIN_DEV_COMPONENT ${PLUGIN_COMPONENT}-dev)
      endif()
    endif()

    list(TRANSFORM PLUGIN_SOURCES PREPEND "${CMAKE_CURRENT_SOURCE_DIR}/")

    set_property(GLOBAL APPEND PROPERTY VPP_PLUGINS_LIST ${name})

    vpp_ego_exec(${name}
      RUN
      OUTPUT ${CMAKE_BINARY_DIR}/${name}.so
      ARGS -plugin ${PLUGIN_SOURCES}
    )

    install(
      FILES ${CMAKE_BINARY_DIR}/${name}.so
      DESTINATION ${VPP_LIBRARY_DIR}/vpp_plugins
      COMPONENT ${PLUGIN_COMPONENT}
    )
  endmacro()

  # Only used by libegovpp main shared library.
  macro(add_ego_vpp_plugin_shared_exec name)
    cmake_parse_arguments(PLUGIN
      ""
      "COMPONENT;DEV_COMPONENT;PACKAGE"
      "SOURCES;SUPPORTED_OS_LIST"
      ${ARGN}
    )
    set(_vpp_exclude_plugin 0)
    if(DEFINED VPP_EXCLUDED_PLUGINS AND NOT VPP_EXCLUDED_PLUGINS STREQUAL "")
      get_property(_vpp_plugins_exclude GLOBAL PROPERTY VPP_PLUGINS_EXCLUDE)
      list(FIND _vpp_plugins_exclude ${name} _vpp_exclude_idx)
      if(NOT _vpp_exclude_idx EQUAL -1)
	list(REMOVE_AT _vpp_plugins_exclude ${_vpp_exclude_idx})
	set_property(GLOBAL PROPERTY VPP_PLUGINS_EXCLUDE "${_vpp_plugins_exclude}")
	set(_vpp_exclude_plugin 1)
      endif()
    endif()
    set_property(GLOBAL APPEND PROPERTY VPP_PLUGINS_ALL_LIST ${name})
    if(DEFINED VPP_PLUGINS AND NOT VPP_PLUGINS STREQUAL "")
      if(VPP_PLUGINS STREQUAL "none")
	return()
      endif()
      get_property(_vpp_plugins_filter GLOBAL PROPERTY VPP_PLUGINS_FILTER)
      list(FIND _vpp_plugins_filter ${name} _vpp_plugin_idx)
      if(_vpp_plugin_idx EQUAL -1)
	return()
      endif()
      list(REMOVE_AT _vpp_plugins_filter ${_vpp_plugin_idx})
      set_property(GLOBAL PROPERTY VPP_PLUGINS_FILTER "${_vpp_plugins_filter}")
    elseif(DEFINED VPP_EXCLUDED_PLUGINS AND NOT VPP_EXCLUDED_PLUGINS STREQUAL "")
      if(_vpp_exclude_plugin)
	return()
      endif()
    endif()
    if (PLUGIN_SUPPORTED_OS_LIST AND NOT ${CMAKE_SYSTEM_NAME} IN_LIST PLUGIN_SUPPORTED_OS_LIST)
      message(WARNING "unsupported OS - ${name} plugin disabled")
      return()
    endif()
    set(plugin_name ${name}_plugin)
    set(api_includes)
    if(NOT PLUGIN_COMPONENT)
      set(PLUGIN_COMPONENT vpp-plugin-core)
    endif()
    if(NOT PLUGIN_DEV_COMPONENT)
      if(NOT VPP_EXTERNAL_PROJECT)
	set(PLUGIN_DEV_COMPONENT vpp-dev)
      else()
	set(PLUGIN_DEV_COMPONENT ${PLUGIN_COMPONENT}-dev)
      endif()
    endif()

    vpp_ego_exec(${name}
      OUTPUT ${CMAKE_BINARY_DIR}/${name}.so
      ARGS -shared-exec ${PLUGIN_PACKAGE}
    )

    set_property(GLOBAL APPEND PROPERTY VPP_PLUGINS_LIST ${name})

    install(
      FILES ${CMAKE_BINARY_DIR}/${name}.so
      DESTINATION ${VPP_LIBRARY_DIR}/vpp_plugins
      COMPONENT ${PLUGIN_COMPONENT}
    )
  endmacro()

  macro(vpp_ego_exec name)
    # Silence warning for ninja generator.
    cmake_policy(SET CMP0116 NEW)

    cmake_parse_arguments(VPP_EGO_EXEC
      "EXCLUDE_FROM_ALL;RUN"
      ""
      "ARGS;CFLAGS;DEPENDS;GOPATH;MAIN;OUTPUT;TAGS"
      ${ARGN}
    )

    if(NOT VPP_EGO_EXEC_OUTPUT)
      set(VPP_EGO_EXEC_OUTPUT ${CMAKE_BINARY_DIR}/${name})
    endif()

    set(compile_args "")

    if (VPP_EGO_EXEC_RUN)
      list(APPEND compile_args "run")
    else()
      list(APPEND compile_args "compile")
    endif()

    # main package
    list(APPEND compile_args -gopath ${CMAKE_CURRENT_SOURCE_DIR}/../libegovpp)

    list(APPEND compile_args -build-dir ${CMAKE_BINARY_DIR}/${name}-ego-build)
    list(APPEND compile_args
      -cgo-cppflags -I${CMAKE_CURRENT_SOURCE_DIR}
      -cgo-cppflags -I${CMAKE_SOURCE_DIR}
      -cgo-cppflags -I${CMAKE_BINARY_DIR}/CMakeFiles)

    # Hard for eliot to get compiler cache correct.
    list(APPEND compile_args -no-cache)

    # -g -O0/-O2
    list(APPEND compile_args ${vpp_ego_flags})

    if(VPP_EGO_EXEC_CFLAGS)
      list(APPEND compile_args ${VPP_EGO_EXEC_CFLAGS})
    else()
      list(APPEND compile_args ${ego_cflags})
    endif()

    if(VPP_EGO_EXEC_TAGS)
      foreach(tag IN LISTS VPP_EGO_EXEC_TAGS)
	list(APPEND compile_args -tags ${tag})
      endforeach()
    endif()

    list(APPEND compile_args -depends ${CMAKE_BINARY_DIR}/${name}.d)
    list(APPEND compile_args ${VPP_EGO_EXEC_ARGS})
    list(APPEND compile_args -o ${VPP_EGO_EXEC_OUTPUT})

    add_custom_command (
      COMMENT "ego building ${name}..."
      OUTPUT ${VPP_EGO_EXEC_OUTPUT}
      DEPFILE ${CMAKE_BINARY_DIR}/${name}.d
      COMMAND ego ${compile_args}
    )

    add_custom_target(${name}_target ALL DEPENDS ${VPP_EGO_EXEC_OUTPUT})
    set_property(
      TARGET ${name}_target
      APPEND
      PROPERTY ADDITIONAL_CLEAN_FILES ${VPP_EGO_EXEC_OUTPUT} ${CMAKE_BINARY_DIR}/${name}_build ${CMAKE_BINARY_DIR}/${name}.d
    )
  endmacro()
endif()
