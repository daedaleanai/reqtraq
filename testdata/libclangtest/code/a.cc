#include <a.hh>
#include <type_traits>

namespace na::nb::nc {

/**
 * \brief even more stuff to do
 * \llr REQ-TEST-SWL-3
 */
void hiddenFunction(const Array<int, 10>&) {}

/**
 * \brief some stuff to do
 * \llr REQ-TEST-SWL-1
 */
void doThings() {}

/**
 * \brief even more stuff to do
 * \llr REQ-TEST-SWL-2
 */
void doMoreThings() {}

// @llr REQ-TEST-SWL-3
void allReqsCovered() {}

// @llr REQ-TEST-SWL-3
using MyType = int;

// @llr REQ-TEST-SWL-3
template <typename T>
concept MyConcept = requires(T t) {
    { t++ } -> std::same_as<int>;
};

}  // namespace na::nb::nc
