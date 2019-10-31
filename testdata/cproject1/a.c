
#include "x/y"

namespace na {
namespace nb {
namespace nc {

namespace {
uint8_t numberOfSegments = 0;
PcieSegment segment;
}  // namespace

// @llr REQ-PROJ-SWH-11
uint8_t System::getNumberOfSegments() {
    return numberOfSegments;
}

// This method does stuff.
// @llr REQ-PROJ-SWL-12
const PcieSegment *System::getSegment(uint8_t i) {
    if (numberOfSegments == 1) {
        return &segment;
    }
    return nullptr;
}

// This method does stuff also.
// @llr REQ-PROJ-SWL-13
// @xlr R-1
void enumerateObjects() {
    io::printf("[system] Scanning for objects\n");

    // Comment.
    numberOfSegments = 1;
}

}  // namespace nc
}  // namespace nb
}  // namespace na
