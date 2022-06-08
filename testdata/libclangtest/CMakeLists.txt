cmake_minimum_required(VERSION 3.16)

project(myproj)

add_executable(myexec code/a.cc)
target_include_directories(myexec PRIVATE code/include)
target_compile_options(myexec PRIVATE -Os)
target_compile_options(myexec PRIVATE -Os)
set_target_properties(myexec PROPERTIES
    CXX_STANDARD 20
    CXX_STANDARD_REQUIRED ON
)

add_executable(myexec_tests test/a/a_test.cc)
target_include_directories(myexec_tests PRIVATE code/include)
target_compile_options(myexec_tests PRIVATE -Os)
set_target_properties(myexec_tests PROPERTIES
    CXX_STANDARD 20
    CXX_STANDARD_REQUIRED ON
)
