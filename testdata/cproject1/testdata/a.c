#include "x/y"

namespace na {
namespace nb {
namespace nc {

namespace {
uint8_t numberOfSegments = 0;
PcieSegment segment;
}  // namespace

// This is a test and is linked to requirements
// @llr REQ-TEST-SWL-13
void testThatSomethingHappens() {}

// This method does stuff.
const PcieSegment *System::getSegment(uint8_t i) {
  if (numberOfSegments == 1) {
    return &segment;
  }
  return nullptr;
}

// This method does stuff also.
// @xlr R-1
void enumerateObjects() {
  io::printf("[system] Scanning for objects\n");

  // Comment.
  numberOfSegments = 1;
}

}  // namespace nc
}  // namespace nb
}  // namespace na
