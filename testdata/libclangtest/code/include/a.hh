#pragma once

#include <utility>

namespace na::nb::nc {

template <typename A, typename B>
struct SomeType {
    A instanceA;
    B instanceB;
};

/**
 * \brief some stuff to do
 * \llr REQ-TEST-SWL-1
 */
void doThings(const SomeType<int, float> &);

/**
 * \brief even more stuff to do
 * \llr REQ-TEST-SWL-2
 */
void doMoreThings(const SomeType<float, int> &);

template <typename T, std::size_t N>
class Array {
    /*
     * This one should not appear, since it is private
     */
    void HiddenMethod();

   public:
    /**
     * \brief Construct array
     * \llr REQ-TEST-SWL-2
     */
    template <typename... Args>
    Array(Args &&...args) : mData{std::forward<Args>(args)...} {}

    /**
     * \brief Return reference to element
     * \llr REQ-TEST-SWL-2
     * \llr REQ-TEST-SWL-12
     */
    T &operator[](std::size_t index) { return mData[index]; }

   private:
    T mData[N];

    /*
     * This one should not appear, since it is private
     */
    void MorePrivateStuff();

   public:
    /*
     * \llr REQ-TEST-SWL-2
     */
    void ButThisIsPublic();
};

struct A {
    /**
     * \llr REQ-TEST-SWL-2
     */
    void StructMethodsArePublicByDefault();

   private:
    void ButCanHavePrivateFunctions();
};

/**
 * \llr REQ-TEST-SWL-2
 */
void JustAFreeFunction();

namespace {
void functionInAnAnonymousNamespace();
}

namespace detail {
void functionInADetailNamespace();
}

/**
 * \llr REQ-TEST-SWL-2
 */
template <typename Iterator, typename Comparator>
void sort(Iterator begin, Iterator end, Comparator c);

/**
 * \llr REQ-TEST-SWL-2
 */
template <typename Iterator>
void sort(Iterator begin, Iterator end) {
    sort(begin, end,
         [](const auto &l, const auto &r) -> bool { return l < r; });
}

template <typename T>
class B final {
    class ShouldNotBeFound final {
        ShouldNotBeFound() = default;
        using Hello = int;

       public:
        explicit ShouldNotBeFound(T &);
        void AlsoNotFound();
    };

   public:
    // @llr REQ-TEST-SWL-2
    void cool();

    // Deleted member
    B() = delete;
};

void JustAFreeFunction();

extern "C" {

/**
 * \llr REQ-TEST-SWL-2
 */
void ExternCFunc();
}

/**
 * \brief This declaration differs from the definition in its LLRs. Therefore
 * reqtraq must flag it.
 * \llr REQ-TEST-SWL-2
 */
void doThings();

class C final {
   public:
    /**
     * \llr REQ-TEST-SWL-2
     */
    ~C();
};

class Abstract {
   public:
    virtual void noImpl() = 0;
    ~Abstract();
};
}  // namespace na::nb::nc
