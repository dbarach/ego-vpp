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

  # Add a target to build
  add_custom_target(build_${name} ALL DEPENDS ${name}.o)

  add_custom_command (OUTPUT ${name}.o
    DEPFILE ${CMAKE_BINARY_DIR}/${name}.d
    COMMAND ego compile -verbose -goroot ${CMAKE_SOURCE_DIR}/plugins/g2f/goroot -tags vpp -link-bitcode -build-dir `pwd` ${EGO_PREFIX_FLAGS} -cgo-exec-path /scratch/fltk -cflags -march=corei7 -cflags -w -cflags -fPIC -depends ${CMAKE_BINARY_DIR}/${name}.d -o `pwd`/${name}.bc ${EGO_PREFIX_PACKAGE} &&
    clang -v -Wall -Wno-override-module -march=corei7 -mtune=corei7-avx -c ${EGO_PREFIX_FLAGS} -fstack-protector -DFORTIFY_SOURCE=2 -maes -mavx2 -fPIC `pwd`/${name}.bc
    COMMENT "Building ${name}.o ..."
    )
endmacro()
