---
name: swiftui-guides
description: "12 SwiftUI best practice guides covering Liquid Glass (iOS 26), navigation, state management, animations, layout, lists, forms, performance, and more. Prevents common LLM mistakes in SwiftUI code generation."
---

# SwiftUI Guides

12 guides for writing modern, production-quality SwiftUI code.

# SwiftUI Animations Reference

Comprehensive guide to SwiftUI animations: basics, transitions, keyframes, phase animators, and custom Animatable conformance.

## Core Concepts

State changes trigger view updates. SwiftUI provides mechanisms to animate these changes.

**Animation Process:**
1. State change triggers view tree re-evaluation
2. SwiftUI compares new tree to current render tree
3. Animatable properties are identified and interpolated (~60 fps)

**Key Characteristics:**
- Animations are additive and cancelable
- Always start from current render tree state
- Blend smoothly when interrupted


## Implicit Animations

Use `.animation(_:value:)` to animate when a specific value changes.

```swift
// GOOD - uses value parameter
Rectangle()
    .frame(width: isExpanded ? 200 : 100, height: 50)
    .animation(.spring, value: isExpanded)
    .onTapGesture { isExpanded.toggle() }

// BAD - deprecated, animates all changes unexpectedly
Rectangle()
    .frame(width: isExpanded ? 200 : 100, height: 50)
    .animation(.spring)  // Deprecated!
```


## Explicit Animations

Use `withAnimation` for event-driven state changes.

```swift
// GOOD - explicit animation
Button("Toggle") {
    withAnimation(.spring) {
        isExpanded.toggle()
    }
}
```

**When to use which:**
- **Implicit**: Animations tied to specific value changes, precise view tree scope
- **Explicit**: Event-driven animations (button taps, gestures)


## Animation Placement

Place animation modifiers after the properties they should animate.

```swift
// GOOD - animation after properties
Rectangle()
    .frame(width: isExpanded ? 200 : 100, height: 50)
    .foregroundStyle(isExpanded ? .blue : .red)
    .animation(.default, value: isExpanded)  // Animates both

// BAD - animation before properties
Rectangle()
    .animation(.default, value: isExpanded)  // Too early!
    .frame(width: isExpanded ? 200 : 100, height: 50)
```


## Selective Animation

```swift
// GOOD - selective animation
Rectangle()
    .frame(width: isExpanded ? 200 : 100, height: 50)
    .animation(.spring, value: isExpanded)  // Animate size
    .foregroundStyle(isExpanded ? .blue : .red)
    .animation(nil, value: isExpanded)  // Don't animate color

// iOS 17+ scoped animation
Rectangle()
    .foregroundStyle(isExpanded ? .blue : .red)  // Not animated
    .animation(.spring) {
        $0.frame(width: isExpanded ? 200 : 100, height: 50)  // Animated
    }
```


## Timing Curves

| Curve | Use Case |
|-------|----------|
| `.spring` | Interactive elements, most UI |
| `.easeInOut` | Appearance changes |
| `.bouncy` | Playful feedback (iOS 17+) |
| `.linear` | Progress indicators only |

```swift
.animation(.default.speed(2.0), value: flag)  // 2x faster
.animation(.default.delay(0.5), value: flag)  // Delayed start
.animation(.default.repeatCount(3, autoreverses: true), value: flag)
```


## Animation Performance

### Prefer Transforms Over Layout

```swift
// GOOD - GPU accelerated transforms
Rectangle()
    .frame(width: 100, height: 100)
    .scaleEffect(isActive ? 1.5 : 1.0)  // Fast
    .offset(x: isActive ? 50 : 0)        // Fast
    .rotationEffect(.degrees(isActive ? 45 : 0))  // Fast

// BAD - layout changes are expensive
Rectangle()
    .frame(width: isActive ? 150 : 100, height: isActive ? 150 : 100)  // Expensive
```

### Narrow Animation Scope

```swift
// GOOD - animation scoped to specific subview
VStack {
    HeaderView()  // Not affected
    ExpandableContent(isExpanded: isExpanded)
        .animation(.spring, value: isExpanded)  // Only this
    FooterView()  // Not affected
}
```

### Avoid Animation in Hot Paths

```swift
// GOOD - gate by threshold
.onPreferenceChange(ScrollOffsetKey.self) { offset in
    let shouldShow = offset.y < -50
    if shouldShow != showTitle {
        withAnimation(.easeOut(duration: 0.2)) {
            showTitle = shouldShow
        }
    }
}
```


## Disabling Animations

```swift
// GOOD - disable with transaction
Text("Count: \(count)")
    .transaction { $0.animation = nil }

// GOOD - disable from parent context
DataView()
    .transaction { $0.disablesAnimations = true }
```


## Transitions

Transitions animate views being inserted or removed from the render tree.

### Critical: Transitions Require Animation Context

```swift
// GOOD - animation outside conditional
VStack {
    Button("Toggle") { showDetail.toggle() }
    if showDetail {
        DetailView()
            .transition(.slide)
    }
}
.animation(.spring, value: showDetail)

// BAD - animation inside conditional (removed with view!)
if showDetail {
    DetailView()
        .transition(.slide)
        .animation(.spring, value: showDetail)  // Won't work on removal!
}
```

### Built-in Transitions

| Transition | Effect |
|------------|--------|
| `.opacity` | Fade in/out (default) |
| `.scale` | Scale up/down |
| `.slide` | Slide from leading edge |
| `.move(edge:)` | Move from specific edge |
| `.offset(x:y:)` | Move by offset amount |

### Combining Transitions

```swift
.transition(.slide.combined(with: .opacity))
```

### Asymmetric Transitions

```swift
// GOOD - different animations for insert/remove
if showCard {
    CardView()
        .transition(
            .asymmetric(
                insertion: .scale.combined(with: .opacity),
                removal: .move(edge: .bottom).combined(with: .opacity)
            )
        )
}
```

### Custom Transitions (iOS 17+)

```swift
struct BlurTransition: Transition {
    var radius: CGFloat

    func body(content: Content, phase: TransitionPhase) -> some View {
        content
            .blur(radius: phase.isIdentity ? 0 : radius)
            .opacity(phase.isIdentity ? 1 : 0)
    }
}
```


## The Animatable Protocol

Enables custom property interpolation during animations.

```swift
struct ShakeModifier: ViewModifier, Animatable {
    var shakeCount: Double

    var animatableData: Double {
        get { shakeCount }
        set { shakeCount = newValue }
    }

    func body(content: Content) -> some View {
        content.offset(x: sin(shakeCount * .pi * 2) * 10)
    }
}
```

### Multiple Properties with AnimatablePair

```swift
struct ComplexModifier: ViewModifier, Animatable {
    var scale: CGFloat
    var rotation: Double

    var animatableData: AnimatablePair<CGFloat, Double> {
        get { AnimatablePair(scale, rotation) }
        set {
            scale = newValue.first
            rotation = newValue.second
        }
    }

    func body(content: Content) -> some View {
        content
            .scaleEffect(scale)
            .rotationEffect(.degrees(rotation))
    }
}
```


## Transactions

The underlying mechanism for all animations in SwiftUI.

```swift
// withAnimation is shorthand for withTransaction
var transaction = Transaction(animation: .default)
withTransaction(transaction) { flag.toggle() }
```

**Implicit animations override explicit animations** (later in view tree wins).


## Phase Animations (iOS 17+)

Cycle through discrete phases automatically.

```swift
// Triggered phase animation
Button("Shake") { trigger += 1 }
    .phaseAnimator(
        [0.0, -10.0, 10.0, -5.0, 5.0, 0.0],
        trigger: trigger
    ) { content, offset in
        content.offset(x: offset)
    }
```

### Enum Phases (Recommended)

```swift
enum BouncePhase: CaseIterable {
    case initial, up, down, settle

    var scale: CGFloat {
        switch self {
        case .initial: 1.0
        case .up: 1.2
        case .down: 0.9
        case .settle: 1.0
        }
    }
}

Circle()
    .phaseAnimator(BouncePhase.allCases, trigger: trigger) { content, phase in
        content.scaleEffect(phase.scale)
    }
```


## Keyframe Animations (iOS 17+)

Precise timing control with exact values at specific times.

```swift
Button("Bounce") { trigger += 1 }
    .keyframeAnimator(
        initialValue: AnimationValues(),
        trigger: trigger
    ) { content, value in
        content
            .scaleEffect(value.scale)
            .offset(y: value.verticalOffset)
    } keyframes: { _ in
        KeyframeTrack(\.scale) {
            SpringKeyframe(1.2, duration: 0.15)
            SpringKeyframe(0.9, duration: 0.1)
            SpringKeyframe(1.0, duration: 0.15)
        }
        KeyframeTrack(\.verticalOffset) {
            LinearKeyframe(-20, duration: 0.15)
            LinearKeyframe(0, duration: 0.25)
        }
    }

struct AnimationValues {
    var scale: CGFloat = 1.0
    var verticalOffset: CGFloat = 0
}
```

| Keyframe Type | Behavior |
|---------------|----------|
| `CubicKeyframe` | Smooth interpolation |
| `LinearKeyframe` | Straight-line interpolation |
| `SpringKeyframe` | Spring physics |
| `MoveKeyframe` | Instant jump (no interpolation) |


## Animation Completion (iOS 17+)

```swift
Button("Animate") {
    withAnimation(.spring) {
        isExpanded.toggle()
    } completion: {
        showNextStep = true
    }
}
```

# SwiftUI Forms & Input Reference

## Form Basics

```swift
struct SettingsView: View {
    @State private var username = ""
    @State private var notificationsEnabled = true
    @State private var selectedColor = Color.blue

    var body: some View {
        Form {
            Section("Profile") {
                TextField("Username", text: $username)
                ColorPicker("Accent Color", selection: $selectedColor)
            }
            Section("Preferences") {
                Toggle("Notifications", isOn: $notificationsEnabled)
            }
        }
    }
}
```

## TextField Patterns

### Styled TextField

```swift
TextField("Email", text: $email)
    .textContentType(.emailAddress)
    .keyboardType(.emailAddress)
    .autocorrectionDisabled()
    .textInputAutocapitalization(.never)

SecureField("Password", text: $password)
    .textContentType(.password)
```

### TextField with Validation

```swift
@State private var email = ""

TextField("Email", text: $email)
    .onChange(of: email) { _, newValue in
        isEmailValid = newValue.contains("@")
    }
    .overlay(alignment: .trailing) {
        if !email.isEmpty {
            Image(systemName: isEmailValid ? "checkmark.circle.fill" : "xmark.circle.fill")
                .foregroundStyle(isEmailValid ? .green : .red)
        }
    }
```

## Picker Patterns

### Segmented Picker

```swift
@State private var selectedTab = 0

Picker("View", selection: $selectedTab) {
    Text("List").tag(0)
    Text("Grid").tag(1)
}
.pickerStyle(.segmented)
```

### Menu Picker

```swift
Picker("Sort By", selection: $sortOrder) {
    Text("Name").tag(SortOrder.name)
    Text("Date").tag(SortOrder.date)
    Text("Size").tag(SortOrder.size)
}
```

### DatePicker

```swift
DatePicker("Due Date", selection: $dueDate, displayedComponents: [.date])
    .datePickerStyle(.compact)
```

## Stepper and Slider

```swift
Stepper("Quantity: \(quantity)", value: $quantity, in: 1...99)

Slider(value: $volume, in: 0...100) {
    Text("Volume")
} minimumValueLabel: {
    Image(systemName: "speaker")
} maximumValueLabel: {
    Image(systemName: "speaker.wave.3")
}
```

## Form Submission

```swift
struct CreateItemView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var isSubmitting = false

    var body: some View {
        NavigationStack {
            Form {
                TextField("Name", text: $name)
            }
            .navigationTitle("New Item")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await submit() }
                    }
                    .disabled(name.isEmpty || isSubmitting)
                }
            }
        }
    }

    private func submit() async {
        isSubmitting = true
        // Save logic
        dismiss()
    }
}
```

# SwiftUI Layout & View Structure Reference

Comprehensive guide to stack layouts, view composition, subview extraction, and layout best practices.

## Relative Layout Over Constants

```swift
// Good - relative to actual layout
GeometryReader { geometry in
    VStack {
        HeaderView()
            .frame(height: geometry.size.height * 0.2)
        ContentView()
    }
}

// Avoid - magic numbers that don't adapt
VStack {
    HeaderView()
        .frame(height: 150)  // Doesn't adapt to different screens
    ContentView()
}
```


## Context-Agnostic Views

Views should work in any context. Never assume presentation style or screen size.

```swift
// Good - adapts to given space
struct ProfileCard: View {
    let user: User

    var body: some View {
        VStack {
            Image(user.avatar)
                .resizable()
                .aspectRatio(contentMode: .fit)
            Text(user.name)
            Spacer()
        }
        .padding()
    }
}

// Avoid - assumes full screen
Image(user.avatar)
    .frame(width: UIScreen.main.bounds.width)  // Wrong!
```


## Own Your Container

Custom views should own static containers but not lazy/repeatable ones.

```swift
// Good - owns static container
struct HeaderView: View {
    var body: some View {
        HStack {
            Image(systemName: "star")
            Text("Title")
            Spacer()
        }
    }
}
```


## View Structure Principles

SwiftUI's diffing algorithm compares view hierarchies to determine what needs updating.

### Prefer Modifiers Over Conditional Views

```swift
// Good - same view, different states
SomeView()
    .opacity(isVisible ? 1 : 0)

// Avoid - creates/destroys view identity
if isVisible {
    SomeView()
}
```

Use conditionals when you truly have **different views**:

```swift
// Correct - fundamentally different views
if isLoggedIn {
    DashboardView()
} else {
    LoginView()
}
```


## Extract Subviews, Not Computed Properties

### The Problem with @ViewBuilder Functions

```swift
// BAD - re-executes complexSection() on every tap
struct ParentView: View {
    @State private var count = 0

    var body: some View {
        VStack {
            Button("Tap: \(count)") { count += 1 }
            complexSection()  // Re-executes every tap!
        }
    }

    @ViewBuilder
    func complexSection() -> some View {
        ForEach(0..<100) { i in
            HStack {
                Image(systemName: "star")
                Text("Item \(i)")
            }
        }
    }
}
```

### The Solution: Separate Structs

```swift
// GOOD - ComplexSection body SKIPPED when its inputs don't change
struct ParentView: View {
    @State private var count = 0

    var body: some View {
        VStack {
            Button("Tap: \(count)") { count += 1 }
            ComplexSection()  // Body skipped during re-evaluation
        }
    }
}

struct ComplexSection: View {
    var body: some View {
        ForEach(0..<100) { i in
            HStack {
                Image(systemName: "star")
                Text("Item \(i)")
            }
        }
    }
}
```


## Container View Pattern

```swift
// BAD - closure prevents SwiftUI from skipping updates
struct MyContainer<Content: View>: View {
    let content: () -> Content
    var body: some View {
        VStack { Text("Header"); content() }
    }
}

// GOOD - view can be compared
struct MyContainer<Content: View>: View {
    @ViewBuilder let content: Content
    var body: some View {
        VStack { Text("Header"); content }
    }
}
```


## ZStack vs overlay/background

Use `ZStack` to **compose multiple peer views** that should be layered together.

Prefer `overlay` / `background` when **decorating a primary view**.

```swift
// GOOD - decoration in overlay
Button("Continue") { }
.overlay(alignment: .trailing) {
    Image(systemName: "lock.fill")
        .padding(.trailing, 8)
}

// GOOD - background shape takes parent size
HStack(spacing: 12) {
    Image(systemName: "tray")
    Text("Inbox")
}
.background {
    Capsule()
        .strokeBorder(.blue, lineWidth: 2)
}
```


## Layout Performance

### Avoid Layout Thrash

```swift
// Bad - deep nesting, excessive layout passes
VStack { HStack { VStack { HStack { Text("Deep") } } } }

// Good - flatter hierarchy
VStack { Text("Shallow"); Text("Structure") }
```

### Minimize GeometryReader (use iOS 17+ alternatives)

```swift
// Good - single geometry reader or containerRelativeFrame
containerRelativeFrame(.horizontal) { width, _ in
    width * 0.8
}
```

### Gate Frequent Geometry Updates

```swift
// Good - gate by threshold
.onPreferenceChange(ViewSizeKey.self) { size in
    let difference = abs(size.width - currentSize.width)
    if difference > 10 { currentSize = size }
}
```


## View Logic and Testability

```swift
// Good - logic in testable model (iOS 17+)
@Observable
@MainActor
final class LoginViewModel {
    var email = ""
    var password = ""
    var isValid: Bool {
        !email.isEmpty && password.count >= 8
    }

    func login() async throws { }
}

struct LoginView: View {
    @State private var viewModel = LoginViewModel()

    var body: some View {
        Form {
            TextField("Email", text: $viewModel.email)
            SecureField("Password", text: $viewModel.password)
            Button("Login") {
                Task { try? await viewModel.login() }
            }
            .disabled(!viewModel.isValid)
        }
    }
}
```


## Action Handlers

```swift
// Good - action references method
struct PublishView: View {
    @State private var viewModel = PublishViewModel()

    var body: some View {
        Button("Publish Project", action: viewModel.handlePublish)
    }
}
```

# SwiftUI Liquid Glass Reference (iOS 26+)

## Overview

Liquid Glass is Apple's new design language introduced in iOS 26. It provides translucent, dynamic surfaces that respond to content and user interaction. This reference covers the native SwiftUI APIs for implementing Liquid Glass effects.

## Availability

All Liquid Glass APIs require iOS 26 or later. Always provide fallbacks:

```swift
if #available(iOS 26, *) {
    // Liquid Glass implementation
} else {
    // Fallback using materials
}
```

## Core APIs

### glassEffect Modifier

The primary modifier for applying glass effects to views:

```swift
.glassEffect(_ style: GlassEffectStyle = .regular, in shape: some Shape = .rect)
```

#### Basic Usage

```swift
Text("Hello")
    .padding()
    .glassEffect()  // Default regular style, rect shape
```

#### With Shape

```swift
Text("Rounded Glass")
    .padding()
    .glassEffect(in: .rect(cornerRadius: 16))

Image(systemName: "star")
    .padding()
    .glassEffect(in: .circle)

Text("Capsule")
    .padding(.horizontal, 20)
    .padding(.vertical, 10)
    .glassEffect(in: .capsule)
```

### GlassEffectStyle

#### Prominence Levels

```swift
.glassEffect(.regular)     // Standard glass appearance
.glassEffect(.prominent)   // More visible, higher contrast
```

#### Tinting

Add color tint to the glass:

```swift
.glassEffect(.regular.tint(.blue))
.glassEffect(.prominent.tint(.red.opacity(0.3)))
```

#### Interactivity

Make glass respond to touch/pointer hover:

```swift
// Interactive glass - responds to user interaction
.glassEffect(.regular.interactive())

// Combined with tint
.glassEffect(.regular.tint(.blue).interactive())
```

**Important**: Only use `.interactive()` on elements that actually respond to user input (buttons, tappable views, focusable elements).

## GlassEffectContainer

Wraps multiple glass elements for proper visual grouping and spacing:

```swift
GlassEffectContainer {
    HStack {
        Button("One") { }
            .glassEffect()
        Button("Two") { }
            .glassEffect()
    }
}
```

### With Spacing

Control the visual spacing between glass elements:

```swift
GlassEffectContainer(spacing: 24) {
    HStack(spacing: 24) {
        GlassChip(icon: "pencil")
        GlassChip(icon: "eraser")
        GlassChip(icon: "trash")
    }
}
```

**Note**: The container's `spacing` parameter should match the actual spacing in your layout for proper glass effect rendering.

## Glass Button Styles

Built-in button styles for glass appearance:

```swift
// Standard glass button
Button("Action") { }
    .buttonStyle(.glass)

// Prominent glass button (higher visibility)
Button("Primary Action") { }
    .buttonStyle(.glassProminent)
```

### Custom Glass Buttons

For more control, apply glass effect manually:

```swift
Button(action: { }) {
    Label("Settings", systemImage: "gear")
        .padding()
}
.glassEffect(.regular.interactive(), in: .capsule)
```

## Morphing Transitions

Create smooth transitions between glass elements using `glassEffectID` and `@Namespace`:

```swift
struct MorphingExample: View {
    @Namespace private var animation
    @State private var isExpanded = false

    var body: some View {
        GlassEffectContainer {
            if isExpanded {
                ExpandedCard()
                    .glassEffect()
                    .glassEffectID("card", in: animation)
            } else {
                CompactCard()
                    .glassEffect()
                    .glassEffectID("card", in: animation)
            }
        }
        .animation(.smooth, value: isExpanded)
    }
}
```

### Requirements for Morphing

1. Both views must have the same `glassEffectID`
2. Use the same `@Namespace`
3. Wrap in `GlassEffectContainer`
4. Apply animation to the container or parent

## Modifier Order

**Critical**: Apply `glassEffect` after layout and visual modifiers:

```swift
// CORRECT order
Text("Label")
    .font(.headline)           // 1. Typography
    .foregroundStyle(.primary) // 2. Color
    .padding()                 // 3. Layout
    .glassEffect()             // 4. Glass effect LAST

// WRONG order - glass applied too early
Text("Label")
    .glassEffect()             // Wrong position
    .padding()
    .font(.headline)
```

## Complete Examples

### Toolbar with Glass Buttons

```swift
struct GlassToolbar: View {
    var body: some View {
        if #available(iOS 26, *) {
            GlassEffectContainer(spacing: 16) {
                HStack(spacing: 16) {
                    ToolbarButton(icon: "pencil", action: { })
                    ToolbarButton(icon: "eraser", action: { })
                    ToolbarButton(icon: "scissors", action: { })
                    Spacer()
                    ToolbarButton(icon: "square.and.arrow.up", action: { })
                }
                .padding(.horizontal)
            }
        } else {
            // Fallback toolbar
            HStack(spacing: 16) {
                // ... fallback implementation
            }
        }
    }
}

struct ToolbarButton: View {
    let icon: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.title2)
                .frame(width: 44, height: 44)
        }
        .glassEffect(.regular.interactive(), in: .circle)
    }
}
```

### Card with Glass Effect

```swift
struct GlassCard: View {
    let title: String
    let subtitle: String

    var body: some View {
        if #available(iOS 26, *) {
            cardContent
                .glassEffect(.regular, in: .rect(cornerRadius: 20))
        } else {
            cardContent
                .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 20))
        }
    }

    private var cardContent: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
            Text(subtitle)
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}
```

### Segmented Control

```swift
struct GlassSegmentedControl: View {
    @Binding var selection: Int
    let options: [String]
    @Namespace private var animation

    var body: some View {
        if #available(iOS 26, *) {
            GlassEffectContainer(spacing: 4) {
                HStack(spacing: 4) {
                    ForEach(options.indices, id: \.self) { index in
                        Button(options[index]) {
                            withAnimation(.smooth) {
                                selection = index
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                        .glassEffect(
                            selection == index ? .prominent.interactive() : .regular.interactive(),
                            in: .capsule
                        )
                        .glassEffectID(selection == index ? "selected" : "option\(index)", in: animation)
                    }
                }
                .padding(4)
            }
        } else {
            Picker("Options", selection: $selection) {
                ForEach(options.indices, id: \.self) { index in
                    Text(options[index]).tag(index)
                }
            }
            .pickerStyle(.segmented)
        }
    }
}
```

## Fallback Strategies

### Using Materials

```swift
if #available(iOS 26, *) {
    content.glassEffect()
} else {
    content.background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 16))
}
```

### Available Materials for Fallback

- `.ultraThinMaterial` - Closest to glass appearance
- `.thinMaterial` - Slightly more opaque
- `.regularMaterial` - Standard blur
- `.thickMaterial` - More opaque
- `.ultraThickMaterial` - Most opaque

### Conditional Modifier Extension

```swift
extension View {
    @ViewBuilder
    func glassEffectWithFallback(
        _ style: GlassEffectStyle = .regular,
        in shape: some Shape = .rect,
        fallbackMaterial: Material = .ultraThinMaterial
    ) -> some View {
        if #available(iOS 26, *) {
            self.glassEffect(style, in: shape)
        } else {
            self.background(fallbackMaterial, in: shape)
        }
    }
}
```

## Best Practices

### Do

- Use `GlassEffectContainer` for grouped glass elements
- Apply glass after layout modifiers
- Use `.interactive()` only on tappable elements
- Match container spacing with layout spacing
- Provide material-based fallbacks for older iOS
- Keep glass shapes consistent within a feature

### Don't

- Apply glass to every element (use sparingly)
- Use `.interactive()` on static content
- Mix different corner radii arbitrarily
- Forget iOS version checks
- Apply glass before padding/frame modifiers
- Nest `GlassEffectContainer` unnecessarily

## Checklist

- [ ] `#available(iOS 26, *)` with fallback
- [ ] `GlassEffectContainer` wraps grouped elements
- [ ] `.glassEffect()` applied after layout modifiers
- [ ] `.interactive()` only on user-interactable elements
- [ ] `glassEffectID` with `@Namespace` for morphing
- [ ] Consistent shapes and spacing across feature
- [ ] Container spacing matches layout spacing
- [ ] Appropriate prominence levels used

# SwiftUI List Patterns Reference

## ForEach Identity and Stability

**Always provide stable identity for `ForEach`.** Never use `.indices` for dynamic content.

```swift
// Good - stable identity via Identifiable
extension User: Identifiable {
    var id: String { userId }
}

ForEach(users) { user in
    UserRow(user: user)
}

// Good - stable identity via keypath
ForEach(users, id: \.userId) { user in
    UserRow(user: user)
}

// Wrong - indices create static content
ForEach(users.indices, id: \.self) { index in
    UserRow(user: users[index])  // Can crash on removal!
}

// Wrong - unstable identity
ForEach(users, id: \.self) { user in
    UserRow(user: user)  // Only works if User is Hashable and stable
}
```

**Critical**: Ensure **constant number of views per element** in `ForEach`:

```swift
// Good - consistent view count
ForEach(items) { item in
    ItemRow(item: item)
}

// Bad - variable view count breaks identity
ForEach(items) { item in
    if item.isSpecial {
        SpecialRow(item: item)
        DetailRow(item: item)
    } else {
        RegularRow(item: item)
    }
}
```

**Avoid inline filtering:**

```swift
// Bad - unstable identity, changes on every update
ForEach(items.filter { $0.isEnabled }) { item in
    ItemRow(item: item)
}

// Good - prefilter and cache
@State private var enabledItems: [Item] = []

var body: some View {
    ForEach(enabledItems) { item in
        ItemRow(item: item)
    }
    .onChange(of: items) { _, newItems in
        enabledItems = newItems.filter { $0.isEnabled }
    }
}
```

**Avoid `AnyView` in list rows:**

```swift
// Bad - hides identity, increases cost
ForEach(items) { item in
    AnyView(item.isSpecial ? SpecialRow(item: item) : RegularRow(item: item))
}

// Good - Create a unified row view
ForEach(items) { item in
    ItemRow(item: item)
}

struct ItemRow: View {
    let item: Item

    var body: some View {
        if item.isSpecial {
            SpecialRow(item: item)
        } else {
            RegularRow(item: item)
        }
    }
}
```

**Why**: Stable identity is critical for performance and animations. Unstable identity causes excessive diffing, broken animations, and potential crashes.

## Enumerated Sequences

**Always convert enumerated sequences to arrays. To be able to use them in a ForEach.**

```swift
let items = ["A", "B", "C"]

// Correct
ForEach(Array(items.enumerated()), id: \.offset) { index, item in
    Text("\(index): \(item)")
}

// Wrong - Doesn't compile, enumerated() isn't an array
ForEach(items.enumerated(), id: \.offset) { index, item in
    Text("\(index): \(item)")
}
```

## List with Custom Styling

```swift
// Remove default background and separators
List(items) { item in
    ItemRow(item: item)
        .listRowInsets(EdgeInsets(top: 8, leading: 16, bottom: 8, trailing: 16))
        .listRowSeparator(.hidden)
}
.listStyle(.plain)
.scrollContentBackground(.hidden)
.background(Color.customBackground)
.environment(\.defaultMinListRowHeight, 1)  // Allows custom row heights
```

## List with Pull-to-Refresh

```swift
List(items) { item in
    ItemRow(item: item)
}
.refreshable {
    await loadItems()
}
```

## Summary Checklist

- [ ] ForEach uses stable identity (never `.indices` for dynamic content)
- [ ] Constant number of views per ForEach element
- [ ] No inline filtering in ForEach (prefilter and cache instead)
- [ ] No `AnyView` in list rows
- [ ] Don't convert enumerated sequences to arrays
- [ ] Use `.refreshable` for pull-to-refresh
- [ ] Custom list styling uses appropriate modifiers

# SwiftUI Media Reference

## PhotosPicker

```swift
import PhotosUI

struct PhotoPickerView: View {
    @State private var selectedItem: PhotosPickerItem?
    @State private var selectedImage: Image?

    var body: some View {
        VStack {
            if let selectedImage {
                selectedImage
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxHeight: 300)
            }

            PhotosPicker("Select Photo", selection: $selectedItem, matching: .images)
        }
        .onChange(of: selectedItem) { _, newItem in
            Task {
                if let data = try? await newItem?.loadTransferable(type: Data.self),
                   let uiImage = UIImage(data: data) {
                    selectedImage = Image(uiImage: uiImage)
                }
            }
        }
    }
}
```

### Multiple Photo Selection

```swift
@State private var selectedItems: [PhotosPickerItem] = []

PhotosPicker("Select Photos", selection: $selectedItems, maxSelectionCount: 5, matching: .images)
```

## MapKit Integration

```swift
import MapKit

struct MapView: View {
    @State private var position: MapCameraPosition = .automatic
    let annotations: [Location]

    var body: some View {
        Map(position: $position) {
            ForEach(annotations) { location in
                Marker(location.name, coordinate: location.coordinate)
            }
        }
        .mapControls {
            MapUserLocationButton()
            MapCompass()
            MapScaleView()
        }
    }
}
```

### Map with Custom Annotations

```swift
Map(position: $position) {
    ForEach(places) { place in
        Annotation(place.name, coordinate: place.coordinate) {
            Image(systemName: "mappin.circle.fill")
                .foregroundStyle(.red)
                .font(.title)
        }
    }
}
```

## Location Services

```swift
import CoreLocation

@Observable
@MainActor
final class LocationManager: NSObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    var location: CLLocation?
    var authorizationStatus: CLAuthorizationStatus = .notDetermined

    override init() {
        super.init()
        manager.delegate = self
    }

    func requestPermission() {
        manager.requestWhenInUseAuthorization()
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        Task { @MainActor in
            location = locations.last
        }
    }

    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        Task { @MainActor in
            authorizationStatus = manager.authorizationStatus
        }
    }
}
```

## Camera Access

```swift
struct CameraView: UIViewControllerRepresentable {
    @Binding var image: UIImage?
    @Environment(\.dismiss) private var dismiss

    func makeUIViewController(context: Context) -> UIImagePickerController {
        let picker = UIImagePickerController()
        picker.sourceType = .camera
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: UIImagePickerController, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    class Coordinator: NSObject, UIImagePickerControllerDelegate, UINavigationControllerDelegate {
        let parent: CameraView
        init(_ parent: CameraView) { self.parent = parent }

        func imagePickerController(_ picker: UIImagePickerController, didFinishPickingMediaWithInfo info: [UIImagePickerController.InfoKey: Any]) {
            parent.image = info[.originalImage] as? UIImage
            parent.dismiss()
        }
    }
}
```

# Modern SwiftUI APIs Reference

## Overview

This reference covers modern SwiftUI API usage patterns and deprecated API replacements. Always use the latest APIs to ensure forward compatibility and access to new features.

## Styling and Appearance

### foregroundStyle() vs foregroundColor()

**Always use `foregroundStyle()` instead of `foregroundColor()`.**

```swift
// Modern (Correct)
Text("Hello")
    .foregroundStyle(.primary)

Image(systemName: "star")
    .foregroundStyle(.blue)

// Legacy (Avoid)
Text("Hello")
    .foregroundColor(.primary)
```

**Why**: `foregroundStyle()` supports hierarchical styles, gradients, and materials, making it more flexible and future-proof.

### clipShape() vs cornerRadius()

**Always use `clipShape(.rect(cornerRadius:))` instead of `cornerRadius()`.**

```swift
// Modern (Correct)
Image("photo")
    .clipShape(.rect(cornerRadius: 12))

VStack {
    // content
}
.clipShape(.rect(cornerRadius: 16))

// Legacy (Avoid)
Image("photo")
    .cornerRadius(12)
```

**Why**: `cornerRadius()` is deprecated. `clipShape()` is more explicit and supports all shape types.

### fontWeight() vs bold()

**Don't apply `fontWeight()` unless there's a good reason. Always use `bold()` for bold text.**

```swift
// Correct
Text("Important")
    .bold()

// Avoid (unless you need a specific weight)
Text("Important")
    .fontWeight(.bold)

// Acceptable (specific weight needed)
Text("Semibold")
    .fontWeight(.semibold)
```

## Navigation

### NavigationStack vs NavigationView

**Always use `NavigationStack` instead of `NavigationView`.**

```swift
// Modern (Correct)
NavigationStack {
    List(items) { item in
        NavigationLink(value: item) {
            Text(item.name)
        }
    }
    .navigationDestination(for: Item.self) { item in
        DetailView(item: item)
    }
}

// Legacy (Avoid)
NavigationView {
    List(items) { item in
        NavigationLink(destination: DetailView(item: item)) {
            Text(item.name)
        }
    }
}
```

### navigationDestination(for:)

**Use `navigationDestination(for:)` for type-safe navigation.**

```swift
struct ContentView: View {
    var body: some View {
        NavigationStack {
            List {
                NavigationLink("Profile", value: Route.profile)
                NavigationLink("Settings", value: Route.settings)
            }
            .navigationDestination(for: Route.self) { route in
                switch route {
                case .profile:
                    ProfileView()
                case .settings:
                    SettingsView()
                }
            }
        }
    }
}

enum Route: Hashable {
    case profile
    case settings
}
```

## Tabs

### Tab API vs tabItem()

**For iOS 18 and later, prefer the `Tab` API over `tabItem()` to access modern tab features, and use availability checks or `tabItem()` for earlier OS versions.**

```swift
// Modern (Correct) - iOS 18+
TabView {
    Tab("Home", systemImage: "house") {
        HomeView()
    }
    
    Tab("Search", systemImage: "magnifyingglass") {
        SearchView()
    }
    
    Tab("Profile", systemImage: "person") {
        ProfileView()
    }
}

// Legacy (Avoid)
TabView {
    HomeView()
        .tabItem {
            Label("Home", systemImage: "house")
        }
}
```

**Important**: When using `Tab(role:)` with roles, you must use the new `Tab { } label: { }` syntax for all tabs. Mixing with `.tabItem()` causes compilation errors.

```swift
// Correct - all tabs use Tab syntax
TabView {
    Tab(role: .search) {
        SearchView()
    } label: {
        Label("Search", systemImage: "magnifyingglass")
    }
    
    Tab {
        HomeView()
    } label: {
        Label("Home", systemImage: "house")
    }
}

// Wrong - mixing Tab and tabItem causes errors
TabView {
    Tab(role: .search) {
        SearchView()
    } label: {
        Label("Search", systemImage: "magnifyingglass")
    }
    
    HomeView()  // Error: can't mix with Tab(role:)
        .tabItem {
            Label("Home", systemImage: "house")
        }
}
```

## Interactions

### Button vs onTapGesture()

**Never use `onTapGesture()` unless you specifically need tap location or tap count. Always use `Button` otherwise.**

```swift
// Correct - standard tap action
Button("Tap me") {
    performAction()
}

// Correct - need tap location
Text("Tap anywhere")
    .onTapGesture { location in
        handleTap(at: location)
    }

// Correct - need tap count
Image("photo")
    .onTapGesture(count: 2) {
        handleDoubleTap()
    }

// Wrong - use Button instead
Text("Tap me")
    .onTapGesture {
        performAction()
    }
```

**Why**: `Button` provides proper accessibility, visual feedback, and semantic meaning. Use `onTapGesture()` only when you need its specific features.

### Button with Images

**Always specify text alongside images in buttons for accessibility.**

```swift
// Correct - includes text label
Button("Add Item", systemImage: "plus") {
    addItem()
}

// Also correct - custom label
Button {
    addItem()
} label: {
    Label("Add Item", systemImage: "plus")
}

// Wrong - image only, no text
Button {
    addItem()
} label: {
    Image(systemName: "plus")
}
```

## Layout and Sizing

### Avoid UIScreen.main.bounds

**Never use `UIScreen.main.bounds` to read available space.**

```swift
// Wrong - uses UIKit, doesn't respect safe areas
let screenWidth = UIScreen.main.bounds.width

// Correct - use GeometryReader
GeometryReader { geometry in
    Text("Width: \(geometry.size.width)")
}

// Better - use containerRelativeFrame (iOS 17+)
Text("Full width")
    .containerRelativeFrame(.horizontal)

// Best - let SwiftUI handle sizing
Text("Auto-sized")
    .frame(maxWidth: .infinity)
```

### GeometryReader Alternatives

> **iOS 17+**: `containerRelativeFrame` and `visualEffect` require iOS 17 or later.

**Don't use `GeometryReader` if a newer alternative works.**

```swift
// Modern - containerRelativeFrame
Image("hero")
    .resizable()
    .containerRelativeFrame(.horizontal) { length, axis in
        length * 0.8
    }

// Modern - visualEffect for position-based effects
Text("Parallax")
    .visualEffect { content, geometry in
        content.offset(y: geometry.frame(in: .global).minY * 0.5)
    }

// Legacy - only use if necessary
GeometryReader { geometry in
    Image("hero")
        .frame(width: geometry.size.width * 0.8)
}
```

## Type Erasure

### Avoid AnyView

**Avoid `AnyView` unless absolutely required.**

```swift
// Prefer - use @ViewBuilder
@ViewBuilder
func content() -> some View {
    if condition {
        Text("Option A")
    } else {
        Image(systemName: "photo")
    }
}

// Avoid - type erasure has performance cost
func content() -> AnyView {
    if condition {
        return AnyView(Text("Option A"))
    } else {
        return AnyView(Image(systemName: "photo"))
    }
}

// Acceptable - when protocol conformance requires it
var body: some View {
    // Complex conditional logic that requires type erasure
}
```

## Styling Best Practices

### Dynamic Type

**Don't force specific font sizes. Prefer Dynamic Type.**

```swift
// Correct - respects user's text size preferences
Text("Title")
    .font(.title)

Text("Body")
    .font(.body)

// Avoid - fixed size doesn't scale
Text("Title")
    .font(.system(size: 24))
```

### UIKit Colors

**Avoid using UIKit colors in SwiftUI code.**

```swift
// Correct - SwiftUI colors
Text("Hello")
    .foregroundStyle(.blue)
    .background(.gray.opacity(0.2))

// Wrong - UIKit colors
Text("Hello")
    .foregroundColor(Color(UIColor.systemBlue))
    .background(Color(UIColor.systemGray))
```

## Static Member Lookup

**Prefer static member lookup to struct instances.**

```swift
// Correct - static member lookup
Circle()
    .fill(.blue)
Button("Action") { }
    .buttonStyle(.borderedProminent)

// Verbose - unnecessary struct instantiation
Circle()
    .fill(Color.blue)
Button("Action") { }
    .buttonStyle(BorderedProminentButtonStyle())
```

## Summary Checklist

- [ ] Use `foregroundStyle()` instead of `foregroundColor()`
- [ ] Use `clipShape(.rect(cornerRadius:))` instead of `cornerRadius()`
- [ ] Use `Tab` API instead of `tabItem()`
- [ ] Use `Button` instead of `onTapGesture()` (unless need location/count)
- [ ] Use `NavigationStack` instead of `NavigationView`
- [ ] Use `navigationDestination(for:)` for type-safe navigation
- [ ] Avoid `AnyView` unless required
- [ ] Avoid `UIScreen.main.bounds`
- [ ] Avoid `GeometryReader` when alternatives exist
- [ ] Use Dynamic Type instead of fixed font sizes
- [ ] Avoid hard-coded padding/spacing unless requested
- [ ] Avoid UIKit colors in SwiftUI
- [ ] Use static member lookup (`.blue` vs `Color.blue`)
- [ ] Include text labels with button images
- [ ] Use `bold()` instead of `fontWeight(.bold)`

# SwiftUI Navigation Reference

Comprehensive guide to NavigationStack, TabView, sheets, fullScreenCover, and routing patterns.

## Pattern Selection Guide

| Pattern | When to Use |
|---------|-------------|
| `NavigationStack` | Hierarchical drill-down (list → detail → edit) |
| `TabView` with `Tab` API | 3+ distinct top-level peer sections |
| `.sheet(item:)` | Creation forms, secondary actions, settings |
| `.fullScreenCover` | Immersive experiences (media player, onboarding) |
| `NavigationStack` + `.sheet` | Most MVPs with 2-4 features |


## NavigationStack

### Type-Safe Navigation

```swift
struct ContentView: View {
    var body: some View {
        NavigationStack {
            List {
                NavigationLink("Profile", value: Route.profile)
                NavigationLink("Settings", value: Route.settings)
            }
            .navigationDestination(for: Route.self) { route in
                switch route {
                case .profile:
                    ProfileView()
                case .settings:
                    SettingsView()
                }
            }
        }
    }
}

enum Route: Hashable {
    case profile
    case settings
}
```

### Programmatic Navigation

```swift
struct ContentView: View {
    @State private var navigationPath = NavigationPath()

    var body: some View {
        NavigationStack(path: $navigationPath) {
            List {
                Button("Go to Detail") {
                    navigationPath.append(DetailRoute.item(id: 1))
                }
            }
            .navigationDestination(for: DetailRoute.self) { route in
                switch route {
                case .item(let id):
                    ItemDetailView(id: id)
                }
            }
        }
    }
}

enum DetailRoute: Hashable {
    case item(id: Int)
}
```


## TabView

Use when the app has 3+ distinct, peer-level sections:

```swift
TabView {
    Tab("Home", systemImage: "house") {
        HomeView()
    }
    Tab("Search", systemImage: "magnifyingglass") {
        SearchView()
    }
    Tab("Profile", systemImage: "person") {
        ProfileView()
    }
}
```


## Sheet Patterns

### Item-Driven Sheets (Preferred)

```swift
// Good - item-driven
@State private var selectedItem: Item?

var body: some View {
    List(items) { item in
        Button(item.name) {
            selectedItem = item
        }
    }
    .sheet(item: $selectedItem) { item in
        ItemDetailSheet(item: item)
    }
}

// Avoid - boolean flag requires separate state
@State private var showSheet = false
@State private var selectedItem: Item?
```

**Why**: `.sheet(item:)` automatically handles presentation state and avoids optional unwrapping.

### Sheets Own Their Actions

Sheets should handle their own dismiss and actions internally.

```swift
struct EditItemSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(DataStore.self) private var store

    let item: Item
    @State private var name: String
    @State private var isSaving = false

    init(item: Item) {
        self.item = item
        _name = State(initialValue: item.name)
    }

    var body: some View {
        NavigationStack {
            Form {
                TextField("Name", text: $name)
            }
            .navigationTitle("Edit Item")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(isSaving ? "Saving..." : "Save") {
                        Task { await save() }
                    }
                    .disabled(isSaving || name.isEmpty)
                }
            }
        }
    }

    private func save() async {
        isSaving = true
        await store.updateItem(item, name: name)
        dismiss()
    }
}
```


## Full Screen Cover

```swift
@State private var showOnboarding = false

.fullScreenCover(isPresented: $showOnboarding) {
    OnboardingView()
}
```


## Popover

```swift
@State private var showPopover = false

Button("Show Popover") {
    showPopover = true
}
.popover(isPresented: $showPopover) {
    PopoverContentView()
        .presentationCompactAdaptation(.popover)
}
```


## Alert with Actions

```swift
.alert("Delete Item?", isPresented: $showAlert) {
    Button("Delete", role: .destructive) { deleteItem() }
    Button("Cancel", role: .cancel) { }
} message: {
    Text("This action cannot be undone.")
}
```


## Confirmation Dialog

```swift
.confirmationDialog("Choose an option", isPresented: $showDialog) {
    Button("Option 1") { handleOption1() }
    Button("Option 2") { handleOption2() }
    Button("Cancel", role: .cancel) { }
}
```


## Type-Safe Routing

Always use `navigationDestination(for:)` for type-safe routing:

```swift
.navigationDestination(for: Note.self) { note in
    NoteDetailView(note: note)
}
.navigationDestination(for: Category.self) { category in
    CategoryView(category: category)
}
```

# SwiftUI Performance Reference

Comprehensive guide to performance optimization, lazy loading, image handling, and concurrency patterns.

## Avoid Redundant State Updates

```swift
// BAD - triggers update even if value unchanged
.onReceive(publisher) { value in
    self.currentValue = value
}

// GOOD - only update when different
.onReceive(publisher) { value in
    if self.currentValue != value {
        self.currentValue = value
    }
}
```


## Optimize Hot Paths

```swift
// GOOD - only update when threshold crossed
.onPreferenceChange(ScrollOffsetKey.self) { offset in
    let shouldShow = offset.y <= -32
    if shouldShow != shouldShowTitle {
        shouldShowTitle = shouldShow
    }
}
```


## Pass Only What Views Need

```swift
// Good - pass specific values
struct SettingsView: View {
    @State private var config = AppConfig()

    var body: some View {
        VStack {
            ThemeSelector(theme: config.theme)
            FontSizeSlider(fontSize: config.fontSize)
        }
    }
}
```


## Equatable Views

For views with expensive bodies:

```swift
struct ExpensiveView: View, Equatable {
    let data: SomeData

    static func == (lhs: Self, rhs: Self) -> Bool {
        lhs.data.id == rhs.data.id
    }

    var body: some View { /* Expensive */ }
}

ExpensiveView(data: data).equatable()
```


## POD Views for Fast Diffing

POD (Plain Old Data) views use `memcmp` for fastest diffing — only simple value types, no property wrappers.

```swift
// POD view - fastest diffing
struct FastView: View {
    let title: String
    let count: Int
    var body: some View { Text("\(title): \(count)") }
}
```

**Advanced**: Wrap expensive non-POD views in POD parent views.


## Lazy Loading

```swift
// GOOD - creates views on demand
ScrollView {
    LazyVStack {
        ForEach(items) { item in
            ExpensiveRow(item: item)
        }
    }
}
```


## Task Cancellation

```swift
struct DataView: View {
    @State private var data: [Item] = []

    var body: some View {
        List(data) { item in Text(item.name) }
        .task {
            data = await fetchData()  // Auto-cancelled on disappear
        }
    }
}
```


## Debug View Updates

```swift
var body: some View {
    let _ = Self._printChanges()  // Prints what caused body to be called
    VStack { Text("Count: \(count)") }
}
```


## Eliminate Unnecessary Dependencies

```swift
// Good - narrow dependency
struct ItemRow: View {
    let item: Item
    let themeColor: Color  // Only depends on what it needs
    var body: some View {
        Text(item.name).foregroundStyle(themeColor)
    }
}
```


## Anti-Patterns

### Creating Objects in Body

```swift
// BAD
var body: some View {
    let formatter = DateFormatter()  // Created every body call!
    return Text(formatter.string(from: date))
}

// GOOD
private static let dateFormatter: DateFormatter = {
    let f = DateFormatter(); f.dateStyle = .long; return f
}()
```

### Heavy Computation in Body

```swift
// BAD - sorts array every body call
List(items.sorted { $0.name < $1.name }) { item in Text(item.name) }

// GOOD - compute in model
@Observable @MainActor
final class ItemsViewModel {
    var items: [Item] = []
    var sortedItems: [Item] { items.sorted { $0.name < $1.name } }
}
```

### Unnecessary State

```swift
// BAD - derived state stored separately
@State private var items: [Item] = []
@State private var itemCount: Int = 0  // Unnecessary!

// GOOD - compute derived values
var itemCount: Int { items.count }
```


## AsyncImage Best Practices

```swift
AsyncImage(url: imageURL) { phase in
    switch phase {
    case .empty:
        ProgressView()
    case .success(let image):
        image
            .resizable()
            .aspectRatio(contentMode: .fit)
    case .failure:
        Image(systemName: "photo")
            .foregroundStyle(.secondary)
    @unknown default:
        EmptyView()
    }
}
.frame(width: 200, height: 200)
```


## Image Downsampling (Optional Optimization)

When you encounter `UIImage(data:)` in scrollable lists or grids:

```swift
struct OptimizedImageView: View {
    let imageData: Data
    let targetSize: CGSize
    @State private var processedImage: UIImage?

    var body: some View {
        Group {
            if let processedImage {
                Image(uiImage: processedImage)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
            } else {
                ProgressView()
            }
        }
        .task {
            processedImage = await decodeAndDownsample(imageData, targetSize: targetSize)
        }
    }

    private func decodeAndDownsample(_ data: Data, targetSize: CGSize) async -> UIImage? {
        await Task.detached {
            guard let source = CGImageSourceCreateWithData(data as CFData, nil) else { return nil }
            let options: [CFString: Any] = [
                kCGImageSourceThumbnailMaxPixelSize: max(targetSize.width, targetSize.height),
                kCGImageSourceCreateThumbnailFromImageAlways: true,
                kCGImageSourceCreateThumbnailWithTransform: true
            ]
            guard let cgImage = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else { return nil }
            return UIImage(cgImage: cgImage)
        }.value
    }
}
```


## SF Symbols

```swift
Image(systemName: "star.fill")
    .foregroundStyle(.yellow)

Image(systemName: "heart.fill")
    .symbolRenderingMode(.multicolor)

// Animated symbols (iOS 17+)
Image(systemName: "antenna.radiowaves.left.and.right")
    .symbolEffect(.variableColor)
```


## Common Performance Issues

- View invalidation storms from broad state changes
- Unstable identity in lists causing excessive diffing
- Heavy work in `body` (formatting, sorting, image decoding)
- Layout thrash from deep stacks or preference chains

When performance issues arise, suggest profiling with Instruments (SwiftUI template).

# SwiftUI ScrollView Patterns Reference

## ScrollView Modifiers

### Hiding Scroll Indicators

**Use `.scrollIndicators(.hidden)` modifier instead of initializer parameter.**

```swift
// Modern (Correct)
ScrollView {
    content
}
.scrollIndicators(.hidden)

// Legacy (Avoid)
ScrollView(showsIndicators: false) {
    content
}
```

## ScrollViewReader for Programmatic Scrolling

**Use `ScrollViewReader` for scroll-to-top, scroll-to-bottom, and anchor-based jumps.**

```swift
struct ChatView: View {
    @State private var messages: [Message] = []
    private let bottomID = "bottom"
    
    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack {
                    ForEach(messages) { message in
                        MessageRow(message: message)
                            .id(message.id)
                    }
                    Color.clear
                        .frame(height: 1)
                        .id(bottomID)
                }
            }
            .onChange(of: messages.count) { _, _ in
                withAnimation {
                    proxy.scrollTo(bottomID, anchor: .bottom)
                }
            }
            .onAppear {
                proxy.scrollTo(bottomID, anchor: .bottom)
            }
        }
    }
}
```

### Scroll-to-Top Pattern

```swift
struct FeedView: View {
    @State private var items: [Item] = []
    @State private var scrollToTop = false
    private let topID = "top"
    
    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack {
                    Color.clear
                        .frame(height: 1)
                        .id(topID)
                    
                    ForEach(items) { item in
                        ItemRow(item: item)
                    }
                }
            }
            .onChange(of: scrollToTop) { _, shouldScroll in
                if shouldScroll {
                    withAnimation {
                        proxy.scrollTo(topID, anchor: .top)
                    }
                    scrollToTop = false
                }
            }
        }
    }
}
```

**Why**: `ScrollViewReader` provides programmatic scroll control with stable anchors. Always use stable IDs and explicit animations.

## Scroll Position Tracking

### Basic Scroll Position

**Avoid** - Storing scroll position directly triggers view updates on every scroll frame:

```swift
// ❌ Bad Practice - causes unnecessary re-renders
struct ContentView: View {
    @State private var scrollPosition: CGFloat = 0

    var body: some View {
        ScrollView {
            content
                .background(
                    GeometryReader { geometry in
                        Color.clear
                            .preference(
                                key: ScrollOffsetPreferenceKey.self,
                                value: geometry.frame(in: .named("scroll")).minY
                            )
                    }
                )
        }
        .coordinateSpace(name: "scroll")
        .onPreferenceChange(ScrollOffsetPreferenceKey.self) { value in
            scrollPosition = value
        }
    }
}
```

**Preferred** - Check scroll position and update a flag based on thresholds for smoother, more efficient scrolling:

```swift
// ✅ Good Practice - only updates state when crossing threshold
struct ContentView: View {
    @State private var startAnimation: Bool = false

    var body: some View {
        ScrollView {
            content
                .background(
                    GeometryReader { geometry in
                        Color.clear
                            .preference(
                                key: ScrollOffsetPreferenceKey.self,
                                value: geometry.frame(in: .named("scroll")).minY
                            )
                    }
                )
        }
        .coordinateSpace(name: "scroll")
        .onPreferenceChange(ScrollOffsetPreferenceKey.self) { value in
            if value < -100 {
                startAnimation = true
            } else {
                startAnimation = false
            }
        }
    }
}

struct ScrollOffsetPreferenceKey: PreferenceKey {
    static var defaultValue: CGFloat = 0
    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}
```

### Scroll-Based Header Visibility

```swift
struct ContentView: View {
    @State private var showHeader = true
    
    var body: some View {
        VStack(spacing: 0) {
            if showHeader {
                HeaderView()
                    .transition(.move(edge: .top))
            }
            
            ScrollView {
                content
                    .background(
                        GeometryReader { geometry in
                            Color.clear
                                .preference(
                                    key: ScrollOffsetPreferenceKey.self,
                                    value: geometry.frame(in: .named("scroll")).minY
                                )
                        }
                    )
            }
            .coordinateSpace(name: "scroll")
            .onPreferenceChange(ScrollOffsetPreferenceKey.self) { offset in
                if offset < -50 { // Scrolling down
                   withAnimation { showHeader = false }
                } else if offset > 50 { // Scrolling up
                  withAnimation { showHeader = true }
                }
            }
        }
    }
}
```

## Scroll Transitions and Effects

> **iOS 17+**: All APIs in this section require iOS 17 or later.

### Scroll-Based Opacity

```swift
struct ParallaxView: View {
    var body: some View {
        ScrollView {
            LazyVStack(spacing: 20) {
                ForEach(items) { item in
                    ItemCard(item: item)
                        .visualEffect { content, geometry in
                            let frame = geometry.frame(in: .scrollView)
                            let distance = min(0, frame.minY)
                            return content
                                .opacity(1 + distance / 200)
                        }
                }
            }
        }
    }
}
```

### Parallax Effect

```swift
struct ParallaxHeader: View {
    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                Image("hero")
                    .resizable()
                    .aspectRatio(contentMode: .fill)
                    .frame(height: 300)
                    .visualEffect { content, geometry in
                        let offset = geometry.frame(in: .scrollView).minY
                        return content
                            .offset(y: offset > 0 ? -offset * 0.5 : 0)
                    }
                    .clipped()
                
                ContentView()
            }
        }
    }
}
```

## Scroll Target Behavior

> **iOS 17+**: All APIs in this section require iOS 17 or later.

### Paging ScrollView

```swift
struct PagingView: View {
    var body: some View {
        ScrollView(.horizontal) {
            LazyHStack(spacing: 0) {
                ForEach(pages) { page in
                    PageView(page: page)
                        .containerRelativeFrame(.horizontal)
                }
            }
            .scrollTargetLayout()
        }
        .scrollTargetBehavior(.paging)
    }
}
```

### Snap to Items

```swift
struct SnapScrollView: View {
    var body: some View {
        ScrollView(.horizontal) {
            LazyHStack(spacing: 16) {
                ForEach(items) { item in
                    ItemCard(item: item)
                        .frame(width: 280)
                }
            }
            .scrollTargetLayout()
        }
        .scrollTargetBehavior(.viewAligned)
        .contentMargins(.horizontal, 20)
    }
}
```

## Summary Checklist

- [ ] Use `.scrollIndicators(.hidden)` instead of initializer parameter
- [ ] Use `ScrollViewReader` with stable IDs for programmatic scrolling
- [ ] Always use explicit animations with `scrollTo()`
- [ ] Use `.visualEffect` for scroll-based visual changes
- [ ] Use `.scrollTargetBehavior(.paging)` for paging behavior
- [ ] Use `.scrollTargetBehavior(.viewAligned)` for snap-to-item behavior
- [ ] Gate frequent scroll position updates by thresholds
- [ ] Use preference keys for custom scroll position tracking

# SwiftUI State Management Reference

## Property Wrapper Selection Guide

| Wrapper | Use When | Notes |
|---------|----------|-------|
| `@State` | Internal view state that triggers updates | Must be `private` |
| `@Binding` | Child view needs to modify parent's state | Don't use for read-only |
| `@Bindable` | iOS 17+: View receives `@Observable` object and needs bindings | For injected observables |
| `let` | Read-only value passed from parent | Simplest option |
| `var` | Read-only value that child observes via `.onChange()` | For reactive reads |

**Legacy (Pre-iOS 17):**
| Wrapper | Use When | Notes |
|---------|----------|-------|
| `@StateObject` | View owns an `ObservableObject` instance | Use `@State` with `@Observable` instead |
| `@ObservedObject` | View receives an `ObservableObject` from outside | Never create inline |

## @State

Always mark `@State` properties as `private`. Use for internal view state that triggers UI updates.

```swift
// Correct
@State private var isAnimating = false
@State private var selectedTab = 0
```

**Why Private?** Marking state as `private` makes it clear what's created by the view versus what's passed in. It also prevents accidentally passing initial values that will be ignored (see "Don't Pass Values as @State" below).

### iOS 17+ with @Observable (Preferred)

**Always prefer `@Observable` over `ObservableObject`.** With iOS 17's `@Observable` macro, use `@State` instead of `@StateObject`:

```swift
@Observable
@MainActor  // Always mark @Observable classes with @MainActor
final class DataModel {
    var name = "Some Name"
    var count = 0
}

struct MyView: View {
    @State private var model = DataModel()  // Use @State, not @StateObject

    var body: some View {
        VStack {
            TextField("Name", text: $model.name)
            Stepper("Count: \(model.count)", value: $model.count)
        }
    }
}
```

**Note**: You may want to mark `@Observable` classes with `@MainActor` to ensure thread safety with SwiftUI, unless your project or package uses Default Actor Isolation set to `MainActor`—in which case, the explicit attribute is redundant and can be omitted.

## @Binding

Use only when child view needs to **modify** parent's state. If child only reads the value, use `let` instead.

```swift
// Parent
struct ParentView: View {
    @State private var isSelected = false

    var body: some View {
        ChildView(isSelected: $isSelected)
    }
}

// Child - will modify the value
struct ChildView: View {
    @Binding var isSelected: Bool

    var body: some View {
        Button("Toggle") {
            isSelected.toggle()
        }
    }
}
```

### When NOT to use @Binding

```swift
// Bad - child only displays, doesn't modify
struct DisplayView: View {
    @Binding var title: String  // Unnecessary
    var body: some View {
        Text(title)
    }
}

// Good - use let for read-only
struct DisplayView: View {
    let title: String
    var body: some View {
        Text(title)
    }
}
```

## @StateObject vs @ObservedObject (Legacy - Pre-iOS 17)

**Note**: These are legacy patterns. Always prefer `@Observable` with `@State` for iOS 17+.

The key distinction is **ownership**:

- `@StateObject`: View **creates and owns** the object
- `@ObservedObject`: View **receives** the object from outside

```swift
// Legacy pattern - use @Observable instead
class MyViewModel: ObservableObject {
    @Published var items: [String] = []
}

// View creates it → @StateObject
struct OwnerView: View {
    @StateObject private var viewModel = MyViewModel()

    var body: some View {
        ChildView(viewModel: viewModel)
    }
}

// View receives it → @ObservedObject
struct ChildView: View {
    @ObservedObject var viewModel: MyViewModel

    var body: some View {
        List(viewModel.items, id: \.self) { Text($0) }
    }
}
```

### Common Mistake

Never create an `ObservableObject` inline with `@ObservedObject`:

```swift
// WRONG - creates new instance on every view update
struct BadView: View {
    @ObservedObject var viewModel = MyViewModel()  // BUG!
}

// CORRECT - owned objects use @StateObject
struct GoodView: View {
    @StateObject private var viewModel = MyViewModel()
}
```

### @StateObject instantiation in View's initializer
If you need to create a @StateObject with initialization parameters in your view's custom initializer, be aware of redundant allocations and hidden side effects.

```swift
// WRONG - creates a new ViewModel instance each time the view's initializer is called
// (which can happen multiple times during SwiftUI's structural identity evaluation)
struct MovieDetailsView: View {
    
    @StateObject private var viewModel: MovieDetailsViewModel
    
    init(movie: Movie) {
        let viewModel = MovieDetailsViewModel(movie: movie)
        _viewModel = StateObject(wrappedValue: viewModel)      
    }
    
    var body: some View {
        // ...
    }
}

// CORRECT - creation in @autoclosure prevents multiple instantiations
struct MovieDetailsView: View {
    
    @StateObject private var viewModel: MovieDetailsViewModel
    
    init(movie: Movie) {
        _viewModel = StateObject(
            wrappedValue: MovieDetailsViewModel(movie: movie)
        )      
    }
    
    var body: some View {
        // ...
    }
}
```

**Modern Alternative**: Use `@Observable` with `@State` instead of `ObservableObject` patterns.

## Don't Pass Values as @State

**Critical**: Never declare passed values as `@State` or `@StateObject`. The value you provide is only an initial value and won't update.

```swift
// Parent
struct ParentView: View {
    @State private var item = Item(name: "Original")
    
    var body: some View {
        ChildView(item: item)
        Button("Change") {
            item.name = "Updated"  // Child won't see this!
        }
    }
}

// Wrong - child ignores updates from parent
struct ChildView: View {
    @State var item: Item  // Accepts initial value only!
    
    var body: some View {
        Text(item.name)  // Shows "Original" forever
    }
}

// Correct - child receives updates
struct ChildView: View {
    let item: Item  // Or @Binding if child needs to modify
    
    var body: some View {
        Text(item.name)  // Updates when parent changes
    }
}
```

**Why**: `@State` and `@StateObject` retain values between view updates. That's their purpose. When a parent passes a new value, the child reuses its existing state.

**Prevention**: Always mark `@State` and `@StateObject` as `private`. This prevents them from appearing in the generated initializer.

## @Bindable (iOS 17+)

Use when receiving an `@Observable` object from outside and needing bindings:

```swift
@Observable
final class UserModel {
    var name = ""
    var email = ""
}

struct ParentView: View {
    @State private var user = UserModel()

    var body: some View {
        EditUserView(user: user)
    }
}

struct EditUserView: View {
    @Bindable var user: UserModel  // Received from parent, needs bindings

    var body: some View {
        Form {
            TextField("Name", text: $user.name)
            TextField("Email", text: $user.email)
        }
    }
}
```

## let vs var for Passed Values

### Use `let` for read-only display

```swift
struct ProfileHeader: View {
    let username: String
    let avatarURL: URL

    var body: some View {
        HStack {
            AsyncImage(url: avatarURL)
            Text(username)
        }
    }
}
```

### Use `var` when reacting to changes with `.onChange()`

```swift
struct ReactiveView: View {
    var externalValue: Int  // Watch with .onChange()
    @State private var displayText = ""

    var body: some View {
        Text(displayText)
            .onChange(of: externalValue) { oldValue, newValue in
                displayText = "Changed from \(oldValue) to \(newValue)"
            }
    }
}
```

## Environment and Preferences

### @Environment

Access environment values provided by SwiftUI or parent views:

```swift
struct MyView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        Button("Done") { dismiss() }
            .foregroundStyle(colorScheme == .dark ? .white : .black)
    }
}
```

### @Environment with @Observable (iOS 17+ - Preferred)

**Always prefer this pattern** for sharing state through the environment:

```swift
@Observable
@MainActor
final class AppState {
    var isLoggedIn = false
}

// Inject
ContentView()
    .environment(AppState())

// Access
struct ChildView: View {
    @Environment(AppState.self) private var appState
}
```

### @EnvironmentObject (Legacy - Pre-iOS 17)

Legacy pattern for sharing observable objects through the environment:

```swift
// Legacy pattern - use @Observable with @Environment instead
class AppState: ObservableObject {
    @Published var isLoggedIn = false
}

// Inject at root
ContentView()
    .environmentObject(AppState())

// Access in child
struct ChildView: View {
    @EnvironmentObject var appState: AppState
}
```

## Decision Flowchart

```
Is this value owned by this view?
├─ YES: Is it a simple value type?
│       ├─ YES → @State private var
│       └─ NO (class):
│           ├─ Use @Observable → @State private var (mark class @MainActor)
│           └─ Legacy ObservableObject → @StateObject private var
│
└─ NO (passed from parent):
    ├─ Does child need to MODIFY it?
    │   ├─ YES → @Binding var
    │   └─ NO: Does child need BINDINGS to its properties?
    │       ├─ YES (@Observable) → @Bindable var
    │       └─ NO: Does child react to changes?
    │           ├─ YES → var + .onChange()
    │           └─ NO → let
    │
    └─ Is it a legacy ObservableObject from parent?
        └─ YES → @ObservedObject var (consider migrating to @Observable)
```

## State Privacy Rules

**All view-owned state should be `private`:**

```swift
// Correct - clear what's created vs passed
struct MyView: View {
    // Created by view - private
    @State private var isExpanded = false
    @State private var viewModel = ViewModel()
    @AppStorage("theme") private var theme = "light"
    @Environment(\.colorScheme) private var colorScheme
    
    // Passed from parent - not private
    let title: String
    @Binding var isSelected: Bool
    @Bindable var user: User
    
    var body: some View {
        // ...
    }
}
```

**Why**: This makes dependencies explicit and improves code completion for the generated initializer.

## Avoid Nested ObservableObject

**Note**: This limitation only applies to `ObservableObject`. `@Observable` fully supports nested observed objects.

```swift
// Avoid - breaks animations and change tracking
class Parent: ObservableObject {
    @Published var child: Child  // Nested ObservableObject
}

class Child: ObservableObject {
    @Published var value: Int
}

// Workaround - pass child directly to views
struct ParentView: View {
    @StateObject private var parent = Parent()
    
    var body: some View {
        ChildView(child: parent.child)  // Pass nested object directly
    }
}

struct ChildView: View {
    @ObservedObject var child: Child
    
    var body: some View {
        Text("\(child.value)")
    }
}
```

**Why**: SwiftUI can't track changes through nested `ObservableObject` properties. Manual workarounds break animations. With `@Observable`, this isn't an issue.

## Key Principles

1. **Always prefer `@Observable` over `ObservableObject`** for new code
2. **Mark `@Observable` classes with `@MainActor` for thread safety (unless using default actor isolation)`**
3. Use `@State` with `@Observable` classes (not `@StateObject`)
4. Use `@Bindable` for injected `@Observable` objects that need bindings
5. **Always mark `@State` and `@StateObject` as `private`**
6. **Never declare passed values as `@State` or `@StateObject`**
7. With `@Observable`, nested objects work fine; with `ObservableObject`, pass nested objects directly to child views

# SwiftUI Text Formatting Reference

## Modern Text Formatting

**Never use C-style `String(format:)` with Text. Always use format parameters.**

## Number Formatting

### Basic Number Formatting

```swift
let value = 42.12345

// Modern (Correct)
Text(value, format: .number.precision(.fractionLength(2)))
// Output: "42.12"

Text(abs(value), format: .number.precision(.fractionLength(2)))
// Output: "42.12" (absolute value)

// Legacy (Avoid)
Text(String(format: "%.2f", abs(value)))
```

### Integer Formatting

```swift
let count = 1234567

// With grouping separator
Text(count, format: .number)
// Output: "1,234,567" (locale-dependent)

// Without grouping
Text(count, format: .number.grouping(.never))
// Output: "1234567"
```

### Decimal Precision

```swift
let price = 19.99

// Fixed decimal places
Text(price, format: .number.precision(.fractionLength(2)))
// Output: "19.99"

// Significant digits
Text(price, format: .number.precision(.significantDigits(3)))
// Output: "20.0"

// Integer-only
Text(price, format: .number.precision(.integerLength(1...)))
// Output: "19"
```

## Currency Formatting

```swift
let price = 19.99

// Correct - with currency code
Text(price, format: .currency(code: "USD"))
// Output: "$19.99"

// With locale
Text(price, format: .currency(code: "EUR").locale(Locale(identifier: "de_DE")))
// Output: "19,99 €"

// Avoid - manual formatting
Text(String(format: "$%.2f", price))
```

## Percentage Formatting

```swift
let percentage = 0.856

// Correct - with precision
Text(percentage, format: .percent.precision(.fractionLength(1)))
// Output: "85.6%"

// Without decimal places
Text(percentage, format: .percent.precision(.fractionLength(0)))
// Output: "86%"

// Avoid - manual calculation
Text(String(format: "%.1f%%", percentage * 100))
```

## Date and Time Formatting

### Date Formatting

```swift
let date = Date()

// Date only
Text(date, format: .dateTime.day().month().year())
// Output: "Jan 23, 2026"

// Full date
Text(date, format: .dateTime.day().month(.wide).year())
// Output: "January 23, 2026"

// Short date
Text(date, style: .date)
// Output: "1/23/26"
```

### Time Formatting

```swift
let date = Date()

// Time only
Text(date, format: .dateTime.hour().minute())
// Output: "2:30 PM"

// With seconds
Text(date, format: .dateTime.hour().minute().second())
// Output: "2:30:45 PM"

// 24-hour format
Text(date, format: .dateTime.hour(.defaultDigits(amPM: .omitted)).minute())
// Output: "14:30"
```

### Relative Date Formatting

```swift
let futureDate = Date().addingTimeInterval(3600)

// Relative formatting
Text(futureDate, style: .relative)
// Output: "in 1 hour"

Text(futureDate, style: .timer)
// Output: "59:59" (counts down)
```

## String Searching and Comparison

### Localized String Comparison

**Use `localizedStandardContains()` for user-input filtering, not `contains()`.**

```swift
let searchText = "café"
let items = ["Café Latte", "Coffee", "Tea"]

// Correct - handles diacritics and case
let filtered = items.filter { $0.localizedStandardContains(searchText) }
// Matches "Café Latte"

// Wrong - exact match only
let filtered = items.filter { $0.contains(searchText) }
// Might not match "Café Latte" depending on normalization
```

**Why**: `localizedStandardContains()` handles case-insensitive, diacritic-insensitive matching appropriate for user-facing search.

### Case-Insensitive Comparison

```swift
let text = "Hello World"
let search = "hello"

// Correct - case-insensitive
if text.localizedCaseInsensitiveContains(search) {
    // Match found
}

// Also correct - for exact comparison
if text.lowercased() == search.lowercased() {
    // Equal
}
```

### Localized Sorting

```swift
let names = ["Zoë", "Zara", "Åsa"]

// Correct - locale-aware sorting
let sorted = names.sorted { $0.localizedStandardCompare($1) == .orderedAscending }
// Output: ["Åsa", "Zara", "Zoë"]

// Wrong - byte-wise sorting
let sorted = names.sorted()
// Output may not be correct for all locales
```

## Attributed Strings

### Basic Attributed Text

```swift
// Using Text concatenation
Text("Hello ")
    .foregroundStyle(.primary)
+ Text("World")
    .foregroundStyle(.blue)
    .bold()

// Using AttributedString
var attributedString = AttributedString("Hello World")
attributedString.foregroundColor = .primary
if let range = attributedString.range(of: "World") {
    attributedString[range].foregroundColor = .blue
    attributedString[range].font = .body.bold()
}
Text(attributedString)
```

### Markdown in Text

```swift
// Simple markdown
Text("This is **bold** and this is *italic*")

// With links
Text("Visit [Apple](https://apple.com) for more info")

// Multiline markdown
Text("""
# Title
This is a paragraph with **bold** text.
- Item 1
- Item 2
""")
```

## Text Measurement

### Measuring Text Height

```swift
// Wrong (Legacy) - GeometryReader trick
struct MeasuredText: View {
    let text: String
    @State private var textHeight: CGFloat = 0
    
    var body: some View {
        Text(text)
            .background(
                GeometryReader { geometry in
                    Color.clear
                        .onAppear {
                            textWidth = geometry.size.height
                        }
                }
            )
    }
}

// Modern (correct)
struct MeasuredText: View {
    let text: String
    @State private var textHeight: CGFloat = 0
    
    var body: some View {
        Text(text)
            .onGeometryChange(for: CGFloat.self) { geometry in
                geometry.size.height
            } action: { newValue in
                textHeight = newValue
            }
    }
}
```

## Summary Checklist

- [ ] Use `.format` parameters with Text instead of `String(format:)`
- [ ] Use `.currency(code:)` for currency formatting
- [ ] Use `.percent` for percentage formatting
- [ ] Use `.dateTime` for date/time formatting
- [ ] Use `localizedStandardContains()` for user-input search
- [ ] Use `localizedStandardCompare()` for locale-aware sorting
- [ ] Use Text concatenation or AttributedString for styled text
- [ ] Use markdown syntax for simple text formatting
- [ ] All formatting respects user's locale and preferences

**Why**: Modern format parameters are type-safe, localization-aware, and integrate better with SwiftUI's text rendering.

