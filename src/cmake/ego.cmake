macro(ego_plugin_object name)
  cmake_parse_arguments(EGO_PREFIX
    ""
    "PACKAGE"
    "FLAGS"
    ${ARGN}
    )
  if(NOT EGO_PREFIX_FLAGS)
    set(EGO_PREFIX_FLAGS "-g;-O0")
  endif()

  # Include the current set of build dependencies
  if (EXISTS ${CMAKE_BINARY_DIR}/${name}_deps.cmake)
    include(${CMAKE_BINARY_DIR}/${name}_deps.cmake)
  endif()
  # Add a target to build
  add_custom_target(build_${name} ALL DEPENDS ${name}.o)

  add_custom_command (OUTPUT ${name}.o
    DEPENDS ${${name}_DEPS}
    COMMAND ego compile -goroot ${CMAKE_SOURCE_DIR}/plugins/g2/goroot -tags vpp -link-bitcode -cmake-deps ${CMAKE_BINARY_DIR}/${name}_new_deps.cmake -build-dir `pwd` ${EGO_PREFIX_FLAGS} -llvm-link llvm-link-13 -cflags -fPIC -o `pwd`/${name}.bc ${EGO_PREFIX_PACKAGE} &&
    clang -Werror -Wall -march=corei7 -mtune=corei7-avx -c ${EGO_PREFIX_FLAGS} -fstack-protector -DFORTIFY_SOURCE=2 -maes -mavx2 -fPIC `pwd`/${name}.bc &&
    diff -q ${CMAKE_BINARY_DIR}/${name}_new_deps.cmake ${CMAKE_BINARY_DIR}/${name}_deps.cmake  \; if [ $$? != 0 ] \; then cp ${CMAKE_BINARY_DIR}/${name}_new_deps.cmake ${CMAKE_BINARY_DIR}/${name}_deps.cmake || true \; touch ${CMAKE_SOURCE_DIR}/CMakeLists.txt \; fi || true
    COMMENT "Building ${name}.o ..."
    )
endmacro()
