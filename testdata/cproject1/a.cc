#include "x/y"

namespace na {
namespace nb {
namespace nc {

namespace {
uint8_t numberOfSegments = 0;
PcieSegment segment;
}  // namespace

// @llr REQ-TEST-SWH-11
uint8_t System::getNumberOfSegments() { return numberOfSegments; }

// This method does stuff.
// @llr REQ-TEST-SWL-12
const PcieSegment *System::getSegment(uint8_t i) {
    if (numberOfSegments == 1) {
        return &segment;
    }
    return nullptr;
}

// This method does stuff also.
// @llr REQ-TEST-SWL-13
// @xlr R-1
void enumerateObjects() {
    auto lambda = []() { io::printf("[system] Scanning for objects\n"); };

    lambda();

    // Comment.
    numberOfSegments = 1;
}

// @llr REQ-TEST-SWL-13, REQ-TEST-SWL-14
int A::operator[](size_t) { return 0; }

}  // namespace nc
}  // namespace nb
}  // namespace na
