---
name: ios-rules
description: "38 battle-tested iOS development rules covering accessibility, navigation, architecture, dark mode, localization, App Review guidelines, and more. Targets the mistakes LLMs actually make when generating Swift/SwiftUI code."
---

# iOS Development Rules

38 rules for writing production-quality iOS apps. Each rule targets common LLM mistakes with concrete fixes.

# Accessibility

REDUCE MOTION:
- Check: @Environment(\.accessibilityReduceMotion) var reduceMotion
- When enabled:
  - Replace .spring() with .easeInOut(duration: 0.2)
  - Replace slide transitions with .opacity
  - Disable auto-playing animations
  - Keep functional animations (progress bars), remove decorative ones
```swift
withAnimation(reduceMotion ? .easeInOut(duration: 0.2) : .spring(response: 0.3)) {
    // state change
}
.transition(reduceMotion ? .opacity : .slide)
```

REDUCE TRANSPARENCY:
- Check: @Environment(\.accessibilityReduceTransparency) var reduceTransparency
- When enabled: use opaque backgrounds instead of materials/blur.
```swift
.background(reduceTransparency ? Color(AppTheme.Colors.surface) : .ultraThinMaterial)
```

VOICEOVER LABELS:
- All interactive elements: .accessibilityLabel("descriptive text").
- Non-obvious actions: .accessibilityHint("Double tap to delete this item").
- Decorative images: .accessibilityHidden(true).
- Informative images: .accessibilityLabel("Profile photo of John").
- Icon-only buttons: MUST have .accessibilityLabel().
```swift
Button(action: addItem) {
    Image(systemName: "plus")
}
.accessibilityLabel("Add new item")
```

GROUPING & COMBINING:
- Related content (icon + label + value): .accessibilityElement(children: .combine).
- Custom read order: .accessibilityElement(children: .ignore) + manual .accessibilityLabel.
- Cards with multiple elements: combine into single accessible element.
```swift
HStack {
    Image(systemName: "heart.fill")
    Text("Favorites")
    Spacer()
    Text("12")
}
.accessibilityElement(children: .combine)
```

ACCESSIBILITY TRAITS:
- Section headers: .accessibilityAddTraits(.isHeader)
- Buttons that play media: .accessibilityAddTraits(.startsMediaSession)
- Summary/aggregate values: .accessibilityAddTraits(.isSummaryElement)
- Selected items: .accessibilityAddTraits(.isSelected)

FOCUS MANAGEMENT:
- Use @FocusState with field enum for form navigation.
- .submitLabel(.next) to show "Next" on keyboard, .submitLabel(.done) for last field.
- Chain fields with .onSubmit { focusedField = .nextField }.
```swift
enum Field: Hashable { case name, email, password }
@FocusState private var focusedField: Field?

TextField("Name", text: $name)
    .focused($focusedField, equals: .name)
    .submitLabel(.next)
    .onSubmit { focusedField = .email }
```

DYNAMIC TYPE:
- System text styles (.body, .headline, etc.) scale automatically.
- NEVER use .font(.system(size:)) — it opts out of Dynamic Type.
- If layout breaks at large sizes: .minimumScaleFactor(0.8) as last resort.
- Test with Xcode Environment Overrides at the largest accessibility size.
- ScrollView wraps content that may overflow at large type sizes.

COLOR & CONTRAST:
- Don't use color alone for status — always pair with icon + text.
- Minimum 4.5:1 contrast for normal text, 3:1 for large text.
- .foregroundStyle(.secondary) for de-emphasized text (maintains adaptive contrast).

ACCESSIBLE CUSTOM CONTROLS:
- Custom sliders/steppers: .accessibilityValue(), .accessibilityAdjustableAction().
- Custom toggles: .accessibilityAddTraits(.isToggle), .accessibilityValue(isOn ? "on" : "off").
- Progress indicators: .accessibilityValue("\(Int(progress * 100)) percent").

# App Clips

APP CLIPS:
SETUP: Requires separate App Clip target (kind: "app_clip" in plan extensions array).
App Clips are a lightweight version of your app for quick, focused tasks.

INFO.PLIST (auto-configured on App Clip target in project.yml):
NSAppClip dict with NSAppClipRequestEphemeralUserNotification and NSAppClipRequestLocationConfirmation is set automatically. No manual configuration needed.

ASSOCIATED DOMAINS (auto-configured in project.yml entitlements):
appclips:{bundleID} and parent-application-identifiers are set automatically.

APP CLIP EXPERIENCE URL: Configure in App Store Connect. Users launch App Clip via NFC, QR code, Maps, etc.

APP CLIP INVOCATION (receive URL):
struct AppClipApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
                .onContinueUserActivity(NSUserActivityTypeBrowsingWeb) { activity in
                    guard let url = activity.webpageURL else { return }
                    // Handle URL: extract parameters, show relevant content
                }
        }
    }
}

SKOverlay (promote full app from within App Clip):
import StoreKit
@Environment(\.requestAppStoreOverlay) var requestOverlay
Button("Get Full App") {
    requestOverlay(AppStoreOverlay.AppClipCompletion(appIdentifier: "YOUR_APP_ID"))
}

CONSTRAINTS:
- App Clip binary must be < 15 MB
- No access to HealthKit, CallKit, SiriKit (use App Intents in full app)
- Limited background modes
- Use @AppStorage for lightweight persistence (no SwiftData)

# App Review

APP REVIEW (StoreKit):
- import StoreKit; @Environment(\.requestReview) var requestReview
- Call requestReview() — Apple handles the dialog
- Trigger after meaningful engagement (launchCount >= 5, key task completed)
- Track with @AppStorage("launchCount"), increment in .onAppear of root view
- Apple limits to 3 prompts/year — never on first launch
- Check: never request immediately after error, crash, or purchase

# Apple Translation

APPLE ON-DEVICE TRANSLATION (Translation framework):
FRAMEWORK: import Translation (iOS 17.4+, on-device, NO internet required, NO API key)

KEY DISTINCTION: Translation is for translating USER CONTENT on demand (e.g. translating a message from French to English). It is NOT for app localization (.strings files). Do not confuse the two.

MODIFIER APPROACH (simplest — shows system translation sheet):
  @State private var showTranslation = false
  Text(userContent)
      .translationPresentation(isPresented: $showTranslation, text: userContent)
  Button("Translate") { showTranslation = true }

PROGRAMMATIC TRANSLATION (TranslationSession):
  @State private var translatedText = ""

  func translateText(_ input: String) async {
      let config = TranslationSession.Configuration(source: .init(identifier: "fr"), target: .init(identifier: "en"))
      let session = TranslationSession(configuration: config)
      do {
          let response = try await session.translate(input)
          translatedText = response.targetText
      } catch {
          // Handle: language pair not supported on device, model not downloaded
      }
  }

  // Call with .task or Button:
  .task { await translateText(originalText) }

SUPPORTED LANGUAGES: Check Translation.supportedLanguages for the device's available language pairs.
AVAILABILITY: Some language pairs require a model download on first use.
NO ENTITLEMENTS NEEDED: Translation framework requires no special entitlements or Info.plist keys.

# Biometrics

BIOMETRIC AUTHENTICATION (Face ID / Touch ID):
- import LocalAuthentication; LAContext().evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics)
- Requires NSFaceIDUsageDescription permission (add CONFIG_CHANGES)
- Check canEvaluatePolicy first; fall back to passcode if biometrics unavailable
- LAContext().biometryType to detect .faceID vs .touchID vs .none
- Always provide manual unlock alternative (PIN/password)

# Camera

CAMERA & PHOTOS:
- PhotosPicker (PhotosUI) for gallery selection — no permissions needed for limited access
- For camera capture: AVCaptureSession + AVCapturePhotoOutput + UIViewControllerRepresentable wrapper
- Camera requires NSCameraUsageDescription permission (add CONFIG_CHANGES)
- Full photo library requires NSPhotoLibraryUsageDescription
- Use @State private var selectedItem: PhotosPickerItem? with .onChange to load
- Load image: try await item.loadTransferable(type: Data.self)

# Charts

SWIFT CHARTS:
- import Charts; use Chart { } container
- BarMark, LineMark, AreaMark, PointMark, RuleMark for data visualization
- .foregroundStyle(by: .value("Category", item.category)) for color coding
- chartXAxis { AxisMarks() }, chartYAxis { AxisMarks() } for custom axis labels
- Extract Chart into a separate computed property to avoid body complexity
- Use .chartScrollableAxes(.horizontal) for large datasets

# Color & Contrast

CONTRAST RATIO REQUIREMENTS (WCAG 2.1 / Apple HIG):
- Normal text (<20pt): minimum 4.5:1 contrast ratio.
- Large text (20pt+ regular or 14pt+ bold): minimum 3:1 contrast ratio.
- Preferred: 7:1 for maximum readability.
- UI components (icons, borders, controls): minimum 3:1 against background.

SEMANTIC COLOR USAGE:
- Red (.red / destructive): delete, remove, error, critical alert.
- Green (.green): success, complete, enabled, positive.
- Orange (.orange): warning, caution, attention needed.
- Blue (.blue): informational, links, primary actions.
- NEVER use red for positive actions or green for destructive actions.

NEVER RELY ON COLOR ALONE:
- Status indicators: color + icon + text label (e.g. green circle + checkmark + "Complete").
- Error fields: red border + error icon + error message text.
- Charts/graphs: use patterns, shapes, or labels alongside color coding.
- Colorblindness: avoid red/green as the sole differentiator — use red/blue or add shapes.

SYSTEM ADAPTIVE COLORS:
- .primary: adapts to light/dark automatically — use for main text.
- .secondary: lighter text for subtitles, metadata.
- Color(.systemBackground): adapts to system appearance.
- These work well for structural elements; use AppTheme for brand colors.

DARK MODE COLOR PAIRING:
- Every custom color must have both light and dark variants.
- Light mode: dark text on light backgrounds.
- Dark mode: light text on dark backgrounds.
- Both variants must independently meet contrast requirements.
- Test both modes — a color that works in light may fail in dark.

APPTHEME COLOR PATTERNS:
- Use Color(light:dark:) extension when app has appearance switching.
- Use plain Color(hex:) when no dark mode support.
- Define semantic tokens: AppTheme.Colors.primary, .surface, .error, .success.
- NEVER use raw hex strings in views — always reference AppTheme tokens.

TEXT ON IMAGES/GRADIENTS:
- Add a dark overlay (.black.opacity(0.4)) before placing white text on images.
- Or use .shadow(color: .black.opacity(0.3), radius: 2) on text.
- Never place light text on light images without contrast treatment.

OPACITY GUIDELINES:
- Avoid text below .opacity(0.6) — fails contrast requirements.
- Disabled state: use .disabled() modifier (auto-handles opacity correctly).
- Placeholder text: use .foregroundStyle(.secondary) instead of manual opacity.

# Component Patterns

BUTTON HIERARCHY (one primary per screen/section):

| Level      | Style                | Use Case                          | Code                                              |
|------------|----------------------|-----------------------------------|----------------------------------------------------|
| Primary    | .borderedProminent   | Main action (Save, Submit, Start) | .buttonStyle(.borderedProminent).controlSize(.large)|
| Secondary  | .bordered            | Alternative action (Cancel, Edit) | .buttonStyle(.bordered)                            |
| Tertiary   | .borderless          | Low-emphasis (Skip, Learn More)   | .buttonStyle(.borderless)                          |
| Destructive| .borderedProminent   | Delete, Remove                    | .buttonStyle(.borderedProminent).tint(.red)        |

- ONE .borderedProminent per screen/section — multiple primaries confuse the user.
- Full-width primary: .controlSize(.large).frame(maxWidth: .infinity).
- ALWAYS use Button() — never .onTapGesture for actions.
- Disabled buttons: .disabled(condition) — SwiftUI auto-handles opacity.

CARD DESIGN PATTERN:
```swift
VStack(alignment: .leading, spacing: AppTheme.Spacing.xSmall) {
    HStack {
        Image(systemName: "icon.name")
            .font(.title3)
            .foregroundStyle(AppTheme.Colors.primary)
        Spacer()
        Text("metadata")
            .font(.caption)
            .foregroundStyle(.secondary)
    }
    Text("Title")
        .font(.headline)
    Text("Description text goes here")
        .font(.subheadline)
        .foregroundStyle(.secondary)
}
.padding(AppTheme.Spacing.medium)
.background(AppTheme.Colors.surface)
.clipShape(RoundedRectangle(cornerRadius: AppTheme.Style.cornerRadius))
.shadow(color: .black.opacity(0.06), radius: 8, y: 4)
```

INPUT FIELD STATES:
- Normal: TextField with .textFieldStyle(.roundedBorder).
- Focused: @FocusState with visual highlight (border color change or underline).
- Error: red border + error message below field.
```swift
TextField("Email", text: $email)
    .textFieldStyle(.roundedBorder)
    .overlay(
        RoundedRectangle(cornerRadius: 8)
            .stroke(emailError != nil ? .red : .clear, lineWidth: 1)
    )
if let error = emailError {
    Text(error)
        .font(.caption)
        .foregroundStyle(.red)
}
```
- Disabled: .disabled(true) — auto grays out.
- Form grouping: use Form or GroupBox for related fields.

LOADING STATES:

| Pattern           | Use Case                          | Code                                    |
|-------------------|-----------------------------------|-----------------------------------------|
| Inline spinner    | Button action, single item        | ProgressView().controlSize(.small)      |
| Full-screen       | Initial data load                 | ProgressView("Loading...")              |
| Pull-to-refresh   | List refresh                      | .refreshable { await refresh() }        |
| Skeleton          | Content placeholder               | .redacted(reason: .placeholder)         |
| Overlay           | Blocking operation                | .overlay { if loading { ProgressView() } } |

- Disable the triggering button while loading to prevent double-taps.
- Show loading for operations > 300ms. Instant operations need no indicator.

BADGE/CHIP PATTERN:
```swift
Text("Label")
    .font(.caption)
    .fontWeight(.medium)
    .padding(.horizontal, 8)
    .padding(.vertical, 4)
    .background(AppTheme.Colors.primary.opacity(0.15))
    .foregroundStyle(AppTheme.Colors.primary)
    .clipShape(Capsule())
```

TOGGLE/SWITCH:
- Use Toggle for binary settings with immediate effect.
- Label must clearly describe the ON state.
- Group related toggles in a Section with a header.

PICKER PATTERNS:
- 2-4 options: Picker with .segmentedStyle.
- 5+ options: Picker with default menu style or NavigationLink to selection list.
- Date selection: DatePicker with appropriate displayedComponents.

EMPTY STATES:
- Always use ContentUnavailableView for empty lists/collections.
- Include: icon (SF Symbol), title, description, and action button if applicable.
- Never show a blank screen — empty state guides the user to the first action.

DIVIDERS:
- Use sparingly — prefer spacing to create visual separation.
- In lists: SwiftUI List provides dividers automatically.
- Custom dividers: Divider() with .padding(.horizontal) for inset style.

# Dark Mode

DARK/LIGHT MODE:
- 3-way picker (system/light/dark) is the standard pattern:
  @AppStorage("appearance") private var appearance: String = "system"
  private var preferredColorScheme: ColorScheme? {
      switch appearance { case "light": return .light; case "dark": return .dark; default: return nil }
  }
  .preferredColorScheme(preferredColorScheme)    // on outermost container in @main app
- CRITICAL: .preferredColorScheme() MUST be in the root @main app, NOT just in the settings view.
- System option: .preferredColorScheme(nil) follows device setting.
- Settings screen: Picker with light/dark/system options writing to @AppStorage("appearance").

ADAPTIVE THEME COLORS (no color assets needed):
- Switch ALL AppTheme palette colors from plain Color(hex:) to Color(light:dark:) with TWO hex values:
  static let background = Color(light: Color(hex: "#F8F9FA"), dark: Color(hex: "#1C1C1E"))
  static let surface = Color(light: Color(hex: "#FFFFFF"), dark: Color(hex: "#2C2C2E"))
- Color(light:dark:) uses UIColor(dynamicProvider:) — reacts to .preferredColorScheme() automatically.
- YOU decide the dark palette based on app mood — user does not specify dark colors.
- Dark palette guidelines: darken backgrounds (#1C1C1E, #2C2C2E), lighten/brighten accents slightly, use Color.primary/Color.secondary for text.
- AppTheme MUST include the Color(light:dark:) extension (see shared constraints).

# Design System Rules

## AppTheme Pattern
Every app **MUST** use a centralized theme with **nested enums** for `Colors`, `Fonts`, and `Spacing`. Do NOT use a flat enum with top-level static properties.

```swift
// REQUIRED — always use nested enums
import SwiftUI

enum AppTheme {
    enum Colors {
        static let accent = Color.blue       // one accent per app
        static let textPrimary = Color.primary
        static let textSecondary = Color.secondary
        static let background = Color(.systemBackground)
        static let surface = Color(.secondarySystemBackground)
        static let cardBackground = Color(.secondarySystemGroupedBackground)
    }

    enum Fonts {
        static let largeTitle = Font.largeTitle
        static let title = Font.title
        static let headline = Font.headline
        static let body = Font.body
        static let caption = Font.caption
    }

    enum Spacing {
        static let small: CGFloat = 8
        static let medium: CGFloat = 16
        static let large: CGFloat = 24
        static let cornerRadius: CGFloat = 12
    }
}
```

```swift
// FORBIDDEN — never use flat structure
enum AppTheme {
    static let accentColor = Color.blue   // ❌ wrong
    static let spacing: CGFloat = 8       // ❌ wrong
}
```

Reference as: `AppTheme.Colors.accent`, `AppTheme.Fonts.headline`, `AppTheme.Spacing.medium`

## Typography
- **System fonts only** — use SwiftUI font styles: `.largeTitle`, `.title`, `.headline`, `.body`, `.caption`
- No custom fonts, no downloaded fonts
- Use `AppTheme.Fonts` for consistent sizing

## Icons (SF Symbols)
- **SF Symbols only** for all icons — required for every list row, button, empty state, and tab
- Reference via `Image(systemName: "symbol.name")`
- Pick domain-appropriate symbols (e.g. "checkmark.circle.fill" for todos, "note.text" for notes, "heart.fill" for favorites)
- Use `.symbolRenderingMode(.hierarchical)` or `.symbolRenderingMode(.palette)` for visual depth
- No custom icon assets unless the app concept specifically requires them

## Colors
- **One accent color** that fits the app's purpose
- Use semantic colors: `.primary`, `.secondary`, `Color(.systemBackground)`
- Do NOT add dark mode support, colorScheme checks, or custom dark/light color handling unless the user explicitly requests it

## Spacing Standards
- **16pt** standard padding (outer margins, section spacing)
- **8pt** compact spacing (between related elements)
- **24pt** large spacing (between major sections)
- Use `AppTheme.Spacing` constants throughout

## Empty States
Every list or collection MUST have an empty state. Use `ContentUnavailableView` (iOS 17+) for a polished look:

```swift
// Required — show when collection is empty
if items.isEmpty {
    ContentUnavailableView(
        "No Notes Yet",
        systemImage: "note.text",
        description: Text("Tap + to create your first note")
    )
} else {
    // Show the list
}
```

For custom empty states, use a styled VStack with SF Symbol + descriptive text:

```swift
VStack(spacing: 16) {
    Image(systemName: "tray")
        .font(.system(size: 48))
        .foregroundStyle(.secondary)
    Text("Nothing here yet")
        .font(.title3)
    Text("Add your first item to get started")
        .font(.subheadline)
        .foregroundStyle(.secondary)
}
```

## Animations
Use subtle, purposeful animations for state changes and list mutations:

```swift
// Toggle/complete actions — spring animation
withAnimation(.spring) {
    item.isComplete.toggle()
}

// List insertions/removals — combine opacity + scale
.transition(.opacity.combined(with: .scale))

// Numeric text changes
.contentTransition(.numericText())

// Filter/tab changes
.animation(.default, value: selectedFilter)
```

Rules:
- **Always** use `withAnimation(.spring)` for toggle/complete state changes
- **Always** add `.transition(.opacity.combined(with: .scale))` for list add/remove
- **Never** add gratuitous motion that slows down interaction
- Keep animations subtle — `.spring` and `.default` curves only

# Feedback States

LOADING PATTERNS:

1. Inline button spinner (action on single element):
```swift
Button {
    Task { await save() }
} label: {
    if isSaving {
        ProgressView()
            .controlSize(.small)
    } else {
        Text("Save")
    }
}
.disabled(isSaving)
```

2. Full-screen loading (initial data load):
```swift
if isLoading {
    ProgressView("Loading...")
} else {
    ContentView()
}
```

3. Skeleton loading (content placeholders):
```swift
ForEach(Item.sampleData) { item in
    ItemRow(item: item)
}
.redacted(reason: .placeholder)
```

4. Pull-to-refresh (list content):
```swift
List { ... }
    .refreshable { await viewModel.refresh() }
```

5. Overlay loading (blocking operation):
```swift
.overlay {
    if isProcessing {
        ZStack {
            Color.black.opacity(0.3)
            ProgressView()
                .controlSize(.large)
                .tint(.white)
        }
        .ignoresSafeArea()
    }
}
```

LOADING RULES:
- Show indicator for operations > 300ms.
- ALWAYS disable the triggering button while loading (prevents double-taps).
- Never block the entire UI for a partial operation — use inline spinner.
- Match loading style to scope: button-level → inline, screen-level → full-screen.

ERROR HANDLING UI:

1. Inline validation (below form fields):
```swift
if let error = emailError {
    HStack(spacing: 4) {
        Image(systemName: "exclamationmark.circle.fill")
        Text(error)
    }
    .font(.caption)
    .foregroundStyle(.red)
}
```

2. Alert for blocking errors (require user acknowledgment):
```swift
.alert("Error", isPresented: $showError) {
    Button("Retry") { Task { await retry() } }
    Button("Cancel", role: .cancel) { }
} message: {
    Text(errorMessage)
}
```

3. Banner for non-blocking errors (dismissible):
```swift
if let error = bannerError {
    HStack {
        Image(systemName: "exclamationmark.triangle.fill")
            .foregroundStyle(.orange)
        Text(error)
            .font(.subheadline)
        Spacer()
        Button("Dismiss") { bannerError = nil }
            .font(.caption)
    }
    .padding(AppTheme.Spacing.small)
    .background(.orange.opacity(0.1))
    .clipShape(RoundedRectangle(cornerRadius: 8))
    .padding(.horizontal, AppTheme.Spacing.medium)
}
```

ERROR HANDLING RULES:
- Inline validation: show immediately as user types or on field blur.
- Alert: use for errors that block progress (network failure, permission denied).
- Banner: use for non-critical errors (sync failed, partial data).
- ALWAYS provide a retry path — never leave users stuck.
- Error messages: describe what happened + what the user can do.

SUCCESS FEEDBACK:
- Haptic: UINotificationFeedbackGenerator().notificationOccurred(.success).
- Visual: brief animation (checkmark, scale bounce, color flash).
- NEVER use modal alert for success — too disruptive.
- Subtle confirmation: toast, inline checkmark, or haptic alone.
```swift
// Brief success animation
withAnimation(.spring(response: 0.3)) {
    showSuccess = true
}
DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
    withAnimation { showSuccess = false }
}
```

DISABLED STATE:
- .disabled(condition) — SwiftUI auto-handles opacity reduction.
- Always explain WHY something is disabled (tooltip, caption text, or label).
- Example: "Fill in all required fields to continue" below a disabled button.
- Don't hide actions — show them disabled with explanation.

NETWORK/SYSTEM ERROR PATTERN (ViewModel):
```swift
@MainActor @Observable
class ItemViewModel {
    var items: [Item] = []
    var isLoading = false
    var error: String?

    func loadItems() async {
        isLoading = true
        error = nil
        do {
            items = try await fetchItems()
        } catch {
            self.error = "Couldn't load items. Pull to refresh to try again."
        }
        isLoading = false
    }
}
```

EMPTY VS ERROR VS LOADING:
- Loading: ProgressView or .redacted skeleton.
- Empty (no data yet): ContentUnavailableView with action to create first item.
- Error (load failed): error message + retry button.
- These are three distinct states — never conflate them.

# File Structure Rules

## Line Limit
- **Target**: 150 lines per file — aim for this
- **Hard limit**: 200 lines — files exceeding 200 lines MUST be split using extensions (`View+Sections.swift`)
- Files between 150-200 lines are acceptable if splitting would hurt readability

## Body as Table of Contents
The `body` property should read like a table of contents — only referencing computed properties, not containing implementation:

```swift
// Good
var body: some View {
    VStack {
        headerSection
        contentSection
        footerSection
    }
}

// Bad — implementation directly in body
var body: some View {
    VStack {
        HStack {
            Image(systemName: "person")
            Text(user.name)
                .font(.headline)
            Spacer()
            Button("Edit") { showEdit = true }
        }
        // ... 80+ more lines
    }
}
```

## Extension Splitting Pattern
When a view grows beyond 150 lines, split into extensions by section:

```swift
// ProfileView.swift — main file
struct ProfileView: View {
    @State var viewModel: ProfileViewModel

    var body: some View {
        ScrollView {
            headerSection
            statsSection
            settingsSection
        }
    }
}

// ProfileView+Sections.swift — extracted sections
extension ProfileView {
    var headerSection: some View { ... }
    var statsSection: some View { ... }
    var settingsSection: some View { ... }
}
```

## Directory Structure
```
Models/              → Data model structs (with static sampleData)
Theme/               → AppTheme.swift only
Features/<Name>/     → Co-locate View + ViewModel (e.g. Features/TodoList/TodoListView.swift)
Features/Common/     → Shared reusable views used by multiple features
App/                 → @main app entry point only
```

- NEVER use flat `Views/`, `ViewModels/`, or `Components/` top-level directories
- Every View and its ViewModel MUST live under `Features/<FeatureName>/`
- Shared components go under `Features/Common/`

## One Type Per File
- Each file contains exactly one primary type (struct, class, or enum)
- Extensions of the same type may live in separate files
- File name matches the type name: `ProfileView.swift` for `struct ProfileView`

## Naming Conventions
- Views: `{Feature}View.swift` (e.g., `NotesListView.swift`)
- ViewModels: `{Feature}ViewModel.swift` (e.g., `NotesListViewModel.swift`)
- Models: `{ModelName}.swift` (e.g., `Note.swift`)
- Theme: `AppTheme.swift`
- App entry: `{AppName}App.swift`

# Forbidden Patterns

## Networking — BANNED
- No `URLSession`, no `Alamofire`, no REST clients
- No API calls of any kind
- No `async let` URL fetches
- The app is fully on-device

## UIKit — AVOID BY DEFAULT
- Prefer SwiftUI-first architecture
- `UIKit` imports are allowed only when a required feature has no viable SwiftUI equivalent
- `UIViewRepresentable` / `UIViewControllerRepresentable` are allowed only as minimal bridges for those required UIKit features
- No Storyboards, no XIBs, no Interface Builder

## Third-Party Packages — BANNED
- No SPM (Swift Package Manager) dependencies
- No CocoaPods
- No Carthage
- Use only Apple-native frameworks

## CoreData — BANNED
- Use **SwiftData** instead of CoreData
- No `NSManagedObject`, no `NSPersistentContainer`
- No `.xcdatamodeld` files

## Authentication & Cloud — BANNED
- No authentication screens, login flows, or token management
- No CloudKit, no iCloud sync
- No push notifications
- No Firebase, no Supabase, no backend services

## Type Re-declarations — BANNED
- **NEVER** re-declare types that exist in other project files
- **NEVER** re-declare types from SwiftUI/Foundation (`Color`, `CGPoint`, `Font`, etc.)
- Import the module or file that defines the type
- Each type must be defined in **exactly one file**

```swift
// BANNED — re-declaring Color enum that SwiftUI already provides
enum Color {
    case red, blue, green
}

// BANNED — re-declaring a model that exists in Models/Note.swift
struct Note {
    var title: String
}

// CORRECT — import and use the existing type
import SwiftUI  // provides Color
// Note is already defined in Models/Note.swift, just reference it
```

# Foundation Models

APPLE ON-DEVICE AI (FoundationModels — iOS 26+):
FRAMEWORK: import FoundationModels

AVAILABILITY CHECK (MANDATORY — model may not be available on all devices):
guard case .available = SystemLanguageModel.default.availability else {
    // Show "This feature requires Apple Intelligence" message
    return
}

BASIC TEXT GENERATION:
let session = LanguageModelSession()
let response = try await session.respond(to: "Summarize this text: \(userText)")
print(response.content)

STREAMING GENERATION:
let stream = session.streamResponse(to: prompt)
for try await partial in stream {
    displayText += partial.text
}

STRUCTURED OUTPUT with @Generable:
@Generable
struct RecipeSuggestion {
    @Guide(description: "Name of the dish") var name: String
    @Guide(description: "Estimated prep time in minutes") var prepTime: Int
    @Guide(description: "Main ingredients") var ingredients: [String]
}

let session = LanguageModelSession()
let recipe: RecipeSuggestion = try await session.respond(
    to: "Suggest a quick pasta dish",
    generating: RecipeSuggestion.self
)

SESSION INSTRUCTIONS (system prompt):
let session = LanguageModelSession(instructions: "You are a helpful cooking assistant. Keep responses concise.")

GUARDRAILS:
- Model output is filtered by Apple's safety system
- No internet required — fully on-device
- Context window is limited (~4K tokens typical) — keep prompts concise
- Use session.respond() for single turns, keep session for multi-turn conversations

# Gestures

STANDARD GESTURE TABLE:

| Gesture    | SwiftUI API           | Use Case                         |
|------------|-----------------------|----------------------------------|
| Tap        | Button()              | Primary actions, navigation      |
| Long press | .contextMenu          | Secondary actions, previews      |
| Swipe      | .swipeActions         | List row actions (delete, edit)  |
| Drag       | .draggable/.dropDest  | Reorder, drag-and-drop           |
| Pull       | .refreshable          | Refresh content                  |
| Pinch      | MagnifyGesture        | Zoom images/maps                 |
| Rotate     | RotateGesture         | Rotate content                   |

BUTTON VS ONTAPGESTURE:
- ALWAYS use Button() for tappable UI elements. Never .onTapGesture on Text/Image/VStack.
- Button provides: accessibility labels, hit testing, highlight states, VoiceOver support.
- .onTapGesture only for non-button interactions (dismiss, background tap).

SWIPE ACTIONS (list rows):
- Trailing side: destructive actions (delete, archive).
- Leading side: positive/toggle actions (pin, favorite, mark read).
- Use .tint() to color-code actions.
- Destructive actions: .role(.destructive) for red styling.
```swift
.swipeActions(edge: .trailing, allowsFullSwipe: true) {
    Button(role: .destructive) {
        deleteItem(item)
    } label: {
        Label("Delete", systemImage: "trash")
    }
}
.swipeActions(edge: .leading) {
    Button {
        toggleFavorite(item)
    } label: {
        Label(item.isFavorite ? "Unfavorite" : "Favorite",
              systemImage: item.isFavorite ? "star.slash" : "star.fill")
    }
    .tint(.yellow)
}
```

CONTEXT MENUS:
- Use for secondary actions on any element (not just list rows).
- Group related actions, use Divider() between groups.
- Destructive actions go LAST with .role(.destructive).
```swift
.contextMenu {
    Button("Edit", systemImage: "pencil") { edit(item) }
    Button("Share", systemImage: "square.and.arrow.up") { share(item) }
    Divider()
    Button("Delete", systemImage: "trash", role: .destructive) { delete(item) }
}
```

GESTURE PRIORITY:
- Default: child gestures take priority over parent.
- .highPriorityGesture(): parent gesture overrides child.
- .simultaneousGesture(): both recognize at the same time.
- Use .simultaneousGesture for scroll + pinch zoom combinations.

HAPTIC FEEDBACK PAIRING:
- Tap actions: UIImpactFeedbackGenerator(style: .light) for subtle confirmation.
- Toggle/switch: UIImpactFeedbackGenerator(style: .medium).
- Destructive action: UINotificationFeedbackGenerator().notificationOccurred(.warning).
- Success completion: UINotificationFeedbackGenerator().notificationOccurred(.success).
- Selection change (picker/list): UISelectionFeedbackGenerator().selectionChanged().
- Drag start/end: UIImpactFeedbackGenerator(style: .medium).

CONFIRMATION FOR DESTRUCTIVE ACTIONS:
- Always confirm before destructive actions (delete, remove, clear all).
- Use .confirmationDialog for choices, .alert for single confirmation.
```swift
.confirmationDialog("Delete Item?", isPresented: $showDelete) {
    Button("Delete", role: .destructive) { delete(item) }
    Button("Cancel", role: .cancel) { }
}
```

PULL-TO-REFRESH:
- Use .refreshable {} on List or ScrollView for refresh.
- SwiftUI handles the indicator automatically.
- The closure should be async — SwiftUI shows/hides spinner based on task completion.

SCROLL GESTURES:
- ScrollView handles scroll automatically.
- .scrollDismissesKeyboard(.interactively) for forms with keyboard.
- .scrollIndicators(.hidden) only when indicators are visually distracting.

# Haptics

HAPTIC FEEDBACK:
- UIImpactFeedbackGenerator(style: .medium).impactOccurred() for simple taps
- UINotificationFeedbackGenerator().notificationOccurred(.success/.warning/.error) for outcomes
- UISelectionFeedbackGenerator().selectionChanged() for selection changes
- CoreHaptics for custom patterns: CHHapticEngine + CHHapticEvent
- Always check CHHapticEngine.capabilitiesForHardware().supportsHaptics
- Prepare generator before use: generator.prepare() for lower latency

# Healthkit

HEALTHKIT:
- import HealthKit; HKHealthStore()
- Check availability: HKHealthStore.isHealthDataAvailable()
- Request auth: healthStore.requestAuthorization(toShare:read:)
- Requires NSHealthShareUsageDescription + NSHealthUpdateUsageDescription (add CONFIG_CHANGES)
- Requires com.apple.developer.healthkit entitlement (add CONFIG_CHANGES)
- Query: HKSampleQuery, HKStatisticsQuery for aggregated data
- Common types: HKQuantityType(.stepCount), .heartRate, .activeEnergyBurned

# Live Activities

LIVE ACTIVITIES (ActivityKit + Dynamic Island):
FRAMEWORK: import ActivityKit

SETUP:
- Requires separate extension target (kind: "live_activity" in plan extensions array)
- NSSupportsLiveActivities: YES is auto-configured on the main app target in project.yml
- AppGroup entitlements are auto-configured for data sharing between app and extension

ATTRIBUTES (define in Shared/ directory so both app and extension compile it):
struct DeliveryAttributes: ActivityAttributes {
    public struct ContentState: Codable, Hashable {
        var status: String       // Mutable: changes during the activity
        var progress: Double
    }
    var orderID: String          // Static: set at start, cannot change
}

STARTING AN ACTIVITY (in app):
let attributes = DeliveryAttributes(orderID: "123")
let initialState = DeliveryAttributes.ContentState(status: "Preparing", progress: 0.0)
let content = ActivityContent(state: initialState, staleDate: nil)
let activity = try Activity.request(attributes: attributes, content: content)

UPDATING AN ACTIVITY:
let updatedState = DeliveryAttributes.ContentState(status: "On the way", progress: 0.6)
let updatedContent = ActivityContent(state: updatedState, staleDate: nil)
await activity.update(updatedContent)

ENDING AN ACTIVITY:
await activity.end(nil, dismissalPolicy: .immediate)

LOCK SCREEN / DYNAMIC ISLAND UI (in extension):
struct DeliveryLiveActivityView: View {
    let context: ActivityViewContext<DeliveryAttributes>
    var body: some View {
        HStack { Text(context.state.status); ProgressView(value: context.state.progress) }
    }
}

struct DeliveryWidget: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: DeliveryAttributes.self) { context in
            DeliveryLiveActivityView(context: context)                  // Lock Screen
        } dynamicIsland: { context in
            DynamicIsland {
                DynamicIslandExpandedRegion(.leading) { Text(context.state.status) }
                DynamicIslandExpandedRegion(.trailing) { ProgressView(value: context.state.progress) }
            } compactLeading: {
                Image(systemName: "bicycle")
            } compactTrailing: {
                Text("\(Int(context.state.progress * 100))%")
            } minimal: {
                ProgressView(value: context.state.progress)
            }
        }
    }
}

MANDATORY FILES (every live activity extension MUST have ALL of these):
1. {Name}Bundle.swift — @main WidgetBundle entry point. Without this, the extension has no entry point → linker error "undefined symbol: _main" → CodeSign failure.
2. LiveActivityWidget.swift — Widget struct with ActivityConfiguration(for: Attributes.self) for Lock Screen and Dynamic Island UI.
3. (Optional) Intents.swift — AppIntents for interactive buttons (complete, skip, etc.).

SHARED TYPES (CRITICAL):
- ActivityAttributes struct MUST be defined in Shared/ directory (NOT in main app's Models/). Both the main app and the live activity extension compile Shared/, so the type is visible to both. Defining it only in the main app causes "Cannot find type in scope" in the extension.

SWIFT 6 CONCURRENCY:
- AppIntent static properties MUST use "static let" (not "static var"). Mutable global state violates Swift 6 concurrency rules.
- LiveActivityService (if @MainActor) → all ViewModels calling it must also be @MainActor.

PUSH-TO-START: Use APNs with activity-update payload to start activities remotely (advanced, requires server).

# Localization

LOCALIZATION (.strings files + RTL/LTR + Language Switching):

FORBIDDEN PATTERNS (CRITICAL — violation = broken app):
- NEVER hardcode translations with if/else or switch on language code. Example of FORBIDDEN code:
  if appLanguage == "ar" { Text("الإعدادات") } else { Text("Settings") }
  switch language { case "ar": return "بحث" default: return "Search" }
- NEVER build a manual translation dictionary/map in code (e.g., let translations = ["en": "Settings", "ar": "الإعدادات"]).
- NEVER use ternary operators to pick translated strings: Text(isArabic ? "الإعدادات" : "Settings").
- These patterns bypass Apple's localization system, break when new languages are added, and ignore the environment locale.
- The ONLY correct approach: use string literals in views — Text("Settings"), Button("Save"), .navigationTitle("Dashboard") — and let Localizable.strings handle translations.
- The deployment pipeline generates .strings files automatically. Your code must ONLY contain English string literals.

.strings FILE GENERATION:
- When localization is requested, generate Resources/{lang}.lproj/Localizable.strings for EACH language.
- File format: standard Apple .strings — one "key" = "translation"; per line.
- KEYS MUST BE THE ENGLISH TEXT ITSELF. Example: "Settings" = "Settings"; (en), "Settings" = "الإعدادات"; (ar). NOT snake_case like "settings_title".
- This means Text("Settings") in code auto-localizes because the key IS the English text.
- English .strings: identity mapping (key = value). Other languages: key = translated value.
- ALL user-facing strings MUST have a key: Text(), Button(), Label(), Toggle(), navigationTitle(), Section(), alert titles/messages, ContentUnavailableView labels, placeholder text.

CRITICAL — localization key usage rules:

Rule 1 — String LITERALS in views are auto-localized:
  Text("settings_title"), .navigationTitle("dashboard_title"), Label("tab_workouts", systemImage: "icon")
  SwiftUI treats these as LocalizedStringKey and looks them up using the environment locale.

Rule 2 — String VARIABLES are NOT auto-localized:
  let key = "settings_title"; Text(key) — shows raw key text. This is the #1 localization bug.
  FIX: Text(LocalizedStringKey(key))

Rule 3 — Computed properties returning keys for display:
  If a switch/computed property returns a key as String (e.g. "metric_steps"), and you pass it to Text(label), it will NOT be localized.
  FIX: Return LocalizedStringKey instead of String from the computed property.

Rule 4 — NEVER use String(localized:) in view parameters:
  Text(String(localized: "key")) resolves against SYSTEM locale, NOT the environment locale. Runtime language switching breaks.

Rule 5 — EVERY key in code must exist in EVERY .strings file. Missing key = raw key shown to user.

- Sample data strings stay as plain String — they are demo content, not translatable.

CONFIG_CHANGES FOR LOCALIZATIONS:
- CONFIG_CHANGES MUST include "localizations": ["en", "ar", "es"] (list of all language codes).
- The client reads this to register knownRegions in the Xcode .pbxproj and create .lproj file references.

LANGUAGE SELECTION & SWITCHING:
- Root view: @AppStorage("appLanguage") private var appLanguage: String = "en"
- Root @main app MUST apply .id(appLanguage) on RootView AND set locale/layoutDirection:
  private var layoutDirection: LayoutDirection {
      ["ar", "he", "fa", "ur"].contains(appLanguage) ? .rightToLeft : .leftToRight
  }
  RootView()
      .id(appLanguage)    // MANDATORY: forces full view rebuild on language change
      .environment(\.locale, Locale(identifier: appLanguage))
      .environment(\.layoutDirection, layoutDirection)
- .id(appLanguage) is CRITICAL — without it, changing language causes mirrored/broken layouts because SwiftUI animates the layout direction change instead of rebuilding. With .id(), the entire view tree is destroyed and recreated cleanly.
- Setting locale ALONE does NOT flip layout to RTL. You MUST set layoutDirection explicitly.
- RTL languages: Arabic (ar), Hebrew (he), Persian/Farsi (fa), Urdu (ur).
- App restart NOT needed — .id(appLanguage) forces a full view rebuild when @AppStorage changes.
- Settings screen: add a language picker (Picker or List with checkmark) that writes to @AppStorage("appLanguage").
- Display language name: Locale(identifier: code).localizedString(forLanguageCode: code) ?? code

RTL / LTR LAYOUT DIRECTION:
- .environment(\.layoutDirection) MUST be set explicitly — .environment(\.locale) does NOT set it automatically.
- Use .leading/.trailing (never .left/.right) for alignment, padding, edges.
- Icons that represent direction (back arrows, chevrons, progress bars) MUST call .flipsForRightToLeftLayoutDirection(true).
- Decorative/universal icons (checkmarks, stars, hearts) must NOT flip.
- Text alignment: always use .leading — SwiftUI resolves it to left or right based on layoutDirection.
- Padding/spacing: always use .leading/.trailing edges, never .left/.right (Edge.Set).

LOCALE-AWARE FORMATTING:
- Dates: use .formatted(date:time:) or Text(date, format:) — automatically adapts to user's locale/calendar.
- Numbers: use .formatted() or Text(number, format: .number) — respects locale decimal/grouping separators.
- Currency: use .formatted(.currency(code:)) — locale-aware symbol placement and formatting.
- Measurements: use Measurement + MeasurementFormatter for locale-appropriate units.
- NEVER manually format dates/numbers with hardcoded separators or patterns.

TESTING RTL IN PREVIEWS:
- Add preview with RTL locale: .environment(\.locale, Locale(identifier: "ar")) and .environment(\.layoutDirection, .rightToLeft)
- Verify: text alignment flips, HStack order reverses, directional icons mirror, padding sides swap.

# Maps

MAPS & LOCATION:
- import MapKit; use Map(position:) { } for SwiftUI maps
- Annotation("Label", coordinate:) { } for custom map pins
- MapCircle, MapPolyline for overlays
- CLLocationManager for user location — requires NSLocationWhenInUseUsageDescription (add CONFIG_CHANGES)
- @State private var position: MapCameraPosition = .automatic
- MKLocalSearch for place search; MKDirections for routing
- .mapControls { MapUserLocationButton(); MapCompass() }

OVERLAY PATTERNS ON MAP VIEWS:
- Map() with .ignoresSafeArea() fills screen edge-to-edge (behind status bar and home indicator)
- For overlays on a map (search bar, header, floating buttons), use .safeAreaInset(edge:) on the Map:
  Map(position: $position) { ... }
      .ignoresSafeArea()
      .safeAreaInset(edge: .top) {
          HeaderView()
              .padding(.horizontal)
              .background(.ultraThinMaterial)
      }
      .safeAreaInset(edge: .bottom) {
          ActionBar()
              .padding()
              .background(.ultraThinMaterial)
      }
- NEVER use ZStack { Map(); VStack { overlay }.padding(.top, 60) } — magic numbers break on different devices
- .mapControls positions built-in controls (compass, user location) inside safe area automatically
- For floating action buttons, use .overlay(alignment: .bottomTrailing) with .padding() — overlay respects safe area by default

# MVVM Architecture Rules

## ViewModel Pattern (In-Memory Default)
Every ViewModel follows this exact pattern:

```swift
import SwiftUI

@Observable
@MainActor
class NotesListViewModel {
    var notes: [Note] = Note.sampleData
    var searchText = ""

    var filteredNotes: [Note] {
        if searchText.isEmpty { return notes }
        return notes.filter { $0.title.localizedCaseInsensitiveContains(searchText) }
    }

    func addNote(title: String) {
        let note = Note(title: title)
        notes.insert(note, at: 0)
    }

    func deleteNote(_ note: Note) {
        notes.removeAll { $0.id == note.id }
    }
}
```

## ViewModel Rules
- Annotate with `@Observable` and `@MainActor`
- ViewModel files contain **ONLY** the `@Observable` class — no other type declarations
- Handle business logic: CRUD, filtering, sorting, validation
- Initialize data arrays from model's `static sampleData` so app looks alive on first launch
- May import `SwiftUI` and `UIKit` when needed for framework types, animation helpers, or required platform bridges
- Must NOT contain Views, `UIView`/`UIViewController` declarations, `body`, or `#Preview`

## View Responsibilities
- Views own `@State` for **local UI state** (sheet presented, text field binding, animation flags)
- Views reference ViewModels via `@State var viewModel = SomeViewModel()`
- Every View file **MUST** include a `#Preview` block with sample data

## Data Access Patterns
| What | Where | How |
|------|-------|-----|
| App data (default) | ViewModel | In-memory arrays initialized from sampleData |
| Simple flags/settings | View or ViewModel | `@AppStorage` |
| Transient UI state | View | `@State` |
| Persistent data (only if user asks) | ViewModel | SwiftData `@Query` / `modelContext` |

# Navigation Pattern Rules

## Pattern Selection Guide

| Pattern | When to Use |
|---------|-------------|
| `NavigationStack` | Hierarchical drill-down (list → detail → edit) |
| `TabView` with `Tab` API | 3+ distinct top-level peer sections |
| `.sheet(item:)` | Creation forms, secondary actions, settings |
| `.fullScreenCover` | Immersive experiences (media player, onboarding) |
| `NavigationStack` + `.sheet` | Most MVPs with 2-4 features |

## NavigationStack
Use for hierarchical navigation with a back button:

```swift
NavigationStack {
    List(items) { item in
        NavigationLink(value: item) {
            ItemRow(item: item)
        }
    }
    .navigationDestination(for: Item.self) { item in
        ItemDetailView(item: item)
    }
    .navigationTitle("Items")
}
```

## TabView with Tab API
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

## Sheets
Choose the right sheet variant based on context:

- **`.sheet(item:)`** — for editing or viewing an existing item (the item drives the sheet)
- **`.sheet(isPresented:)`** — acceptable for creation forms and simple actions (no item yet)

```swift
// Editing/viewing an existing item — use item-driven
@State private var editingItem: Item?

.sheet(item: $editingItem) { item in
    EditItemView(item: item)
}

// Creating a new item — isPresented is fine
@State private var showAddItem = false

.sheet(isPresented: $showAddItem) {
    AddItemView()
}
```

## Full Screen Cover
Use for immersive content that should cover the entire screen:

```swift
@State private var showOnboarding = false

.fullScreenCover(isPresented: $showOnboarding) {
    OnboardingView()
}
```

## Type-Safe Routing
Always use `navigationDestination(for:)` for type-safe routing:

```swift
// Define route types
.navigationDestination(for: Note.self) { note in
    NoteDetailView(note: note)
}
.navigationDestination(for: Category.self) { category in
    CategoryView(category: category)
}
```

# Notification Service

NOTIFICATION SERVICE EXTENSION:
SETUP: Requires separate extension target (kind: "notification_service" in plan extensions array).
Used to modify rich push notification content before display (add images, decrypt payload, etc.).

PRINCIPAL CLASS (in extension):
class NotificationService: UNNotificationServiceExtension {
    var contentHandler: ((UNNotificationContent) -> Void)?
    var bestAttemptContent: UNMutableNotificationContent?

    override func didReceive(_ request: UNNotificationRequest,
                             withContentHandler contentHandler: @escaping (UNNotificationContent) -> Void) {
        self.contentHandler = contentHandler
        bestAttemptContent = (request.content.mutableCopy() as? UNMutableNotificationContent)

        guard let bestAttemptContent else { contentHandler(request.content); return }

        // Modify content here — e.g., download and attach image
        if let urlString = request.content.userInfo["image_url"] as? String,
           let url = URL(string: urlString) {
            downloadAndAttach(url: url, to: bestAttemptContent) { modified in
                contentHandler(modified)
            }
        } else {
            bestAttemptContent.title = "[Modified] " + bestAttemptContent.title
            contentHandler(bestAttemptContent)
        }
    }

    override func serviceExtensionTimeWillExpire() {
        // Called just before extension is killed — deliver best attempt
        if let contentHandler, let bestAttemptContent {
            contentHandler(bestAttemptContent)
        }
    }
}

ADDING ATTACHMENT:
func downloadAndAttach(url: URL, to content: UNMutableNotificationContent, completion: @escaping (UNNotificationContent) -> Void) {
    URLSession.shared.downloadTask(with: url) { localURL, _, _ in
        if let localURL, let attachment = try? UNNotificationAttachment(identifier: "image", url: localURL) {
            content.attachments = [attachment]
        }
        completion(content)
    }.resume()
}

# Notifications

NOTIFICATIONS (LOCAL — no server needed):

FRAMEWORK: UserNotifications — UNUserNotificationCenter.current()

PERMISSION STATES (CRITICAL — must handle ALL four states):
1. NOT DETERMINED: Call .requestAuthorization(options: [.alert, .badge, .sound]). System shows dialog.
2. AUTHORIZED: Schedule notifications normally.
3. DENIED: App CANNOT re-request permission. Must redirect user to System Settings.
4. PROVISIONAL: Quiet notifications — treat as authorized.

PERMISSION IS SYSTEM-CONTROLLED (CRITICAL):
- Notification authorization is owned by the SYSTEM, not the app.
- Once the user denies, the app has NO way to re-request — only the user can re-enable in System Settings.
- To open System Settings: UIApplication.shared.open(URL(string: UIApplication.openSettingsURLString)!)

UI THAT DISPLAYS NOTIFICATION STATUS (in ANY view — must handle ALL three states):
- .notDetermined: Show an "Enable Notifications" button that calls requestAuthorization(options: [.alert, .badge, .sound]). After the system dialog, re-check status and update UI.
- .authorized / .provisional: Show enabled state (e.g. "bell" icon, "Notifications are enabled" text). This is read-only — the user can disable in System Settings.
- .denied: Show disabled state ("bell.slash" icon) with an actionable "Open Settings" button that calls UIApplication.shared.open(URL(string: UIApplication.openSettingsURLString)!). Show helper text explaining the user must enable in System Settings.
- NEVER use a writable Toggle for notification permission — the app cannot grant/revoke it programmatically. Use buttons with state-specific actions instead.

RE-CHECK ON FOREGROUND (CRITICAL):
Any view displaying notification status MUST re-check when the app returns to foreground.
The user may have changed permissions in System Settings while the app was backgrounded.

  .onReceive(NotificationCenter.default.publisher(for: UIApplication.willEnterForegroundNotification)) { _ in
      Task {
          let settings = await UNUserNotificationCenter.current().notificationSettings()
          isNotificationsEnabled = (settings.authorizationStatus == .authorized || settings.authorizationStatus == .provisional)
      }
  }

SCHEDULING:
- Local: UNMutableNotificationContent + UNNotificationRequest
- Time-based: UNTimeIntervalNotificationTrigger(timeInterval:repeats:)
- Calendar-based: UNCalendarNotificationTrigger(dateMatching:repeats:)
- Default to local notifications unless user explicitly asks for remote/push.

BADGE COUNT:
- Set via UNMutableNotificationContent().badge = NSNumber(value: count)
- Clear on app open: UIApplication.shared.applicationIconBadgeNumber = 0

# Safari Extension

SAFARI WEB EXTENSION:
SETUP: Requires separate extension target (kind: "safari" in plan extensions array).
Safari Web Extensions use web technologies (JS/HTML/CSS) plus a native Swift wrapper.

SWIFT EXTENSION HANDLER (optional — for native communication):
class SafariExtensionHandler: SFSafariExtensionHandler {
    // Called when toolbar button is clicked
    override func toolbarItemClicked(in window: SFSafariWindow) {
        window.getActiveTab { tab in
            tab?.getActivePage { page in
                page?.dispatchMessageToScript(withName: "buttonClicked", userInfo: [:])
            }
        }
    }

    // Receive messages from JavaScript content scripts
    override func messageReceived(withName messageName: String, from page: SFSafariPage, userInfo: [String: Any]?) {
        // Handle message from JS: page.dispatchMessageToExtension(...)
    }
}

INFO.PLIST KEYS (extension):
NSExtensionPrincipalClass: $(PRODUCT_MODULE_NAME).SafariExtensionHandler
NSExtensionAttributes:
  SFSafariWebsiteAccess: { Level: All }

WEB EXTENSION RESOURCES (in extension bundle):
- manifest.json (Web Extension manifest v3)
- background.js (service worker)
- content.js (injected into pages)
- popup.html (toolbar popup)

ENABLE IN SAFARI: User must enable in Safari → Settings → Extensions.

# Share Extension

SHARE EXTENSION:
SETUP: Requires separate extension target (kind: "share" in plan extensions array).
The extension receives shared content (URLs, text, images) from other apps via the share sheet.

PRINCIPAL CLASS (in extension target):
class ShareViewController: SLComposeServiceViewController {
    override func isContentValid() -> Bool {
        return contentText.count > 0    // Validate before enabling Post button
    }

    override func didSelectPost() {
        // Access shared items
        guard let item = extensionContext?.inputItems.first as? NSExtensionItem,
              let provider = item.attachments?.first else {
            extensionContext?.completeRequest(returningItems: [], completionHandler: nil)
            return
        }

        if provider.hasItemConformingToTypeIdentifier("public.url") {
            provider.loadItem(forTypeIdentifier: "public.url") { [weak self] url, _ in
                // Handle URL
                self?.extensionContext?.completeRequest(returningItems: [], completionHandler: nil)
            }
        }
    }

    override func configurationItems() -> [Any]! {
        return []    // Return SLComposeSheetConfigurationItem array for optional UI
    }
}

INFO.PLIST KEYS (in extension's Info.plist via XcodeGen):
NSExtensionPrincipalClass: $(PRODUCT_MODULE_NAME).ShareViewController
NSExtensionActivationRule:
  NSExtensionActivationSupportsWebURLWithMaxCount: 1
  NSExtensionActivationSupportsText: true

APP GROUP: Use AppGroup entitlement to share data between main app and extension.

# Siri Intents

SIRI VOICE COMMANDS (App Intents):
FRAMEWORK: import AppIntents (iOS 16+, replaces legacy SiriKit Intents)

BASIC INTENT:
struct OpenNoteIntent: AppIntent {
    static var title: LocalizedStringResource = "Open Note"
    static var description = IntentDescription("Opens a specific note in the app")

    @Parameter(title: "Note Name")
    var noteName: String

    func perform() async throws -> some IntentResult & ProvidesDialog {
        // Navigate to note or perform action
        return .result(dialog: "Opening \(noteName)")
    }
}

APP SHORTCUTS (makes intent discoverable via Siri without user setup):
struct MyAppShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(
            intent: OpenNoteIntent(),
            phrases: ["Open \(\.$noteName) in \(.applicationName)", "Show note \(\.$noteName)"],
            shortTitle: "Open Note",
            systemImageName: "note.text"
        )
    }
}

ENTITLEMENT: Add com.apple.developer.siri entitlement (CONFIG_CHANGES entitlements key).
NO SEPARATE TARGET: App Intents run in-process — no extension target needed.
DONATION: AppShortcutsProvider automatically donates shortcuts to Siri.

# Spacing & Layout

8-POINT GRID SYSTEM:
All spacing values must be multiples of 4pt. Standard scale:

| Token             | Value | Use Case                                    |
|-------------------|-------|---------------------------------------------|
| Spacing.xxSmall   | 4pt   | Icon-to-label gap, tight inline spacing     |
| Spacing.xSmall    | 8pt   | Between related items (label + value)        |
| Spacing.small     | 12pt  | List row internal padding                    |
| Spacing.medium    | 16pt  | Standard padding, outer margins              |
| Spacing.large     | 24pt  | Section spacing, card-to-card gap            |
| Spacing.xLarge    | 32pt  | Major section breaks                         |
| Spacing.xxLarge   | 48pt  | Screen-level top/bottom breathing room       |

NEVER use arbitrary values (5, 7, 10, 13, 15, 18, 25, etc.). Snap to the grid.

STANDARD MARGINS:
- Outer screen margins: 16pt (.padding(.horizontal, AppTheme.Spacing.medium)).
- Card internal padding: 16pt (.padding(AppTheme.Spacing.medium)).
- Section spacing (between groups): 24pt.
- List row vertical padding: 12pt.
- Toolbar/header spacing: 16pt horizontal, 8pt vertical.

TAP TARGET SIZES:
- Minimum: 44x44pt for ALL interactive elements (Apple HIG requirement).
- If visual element is smaller (e.g. 24x24 icon), expand hit area:
  .frame(minWidth: 44, minHeight: 44)
  .contentShape(Rectangle())
- Inline buttons in text: ensure surrounding padding creates 44pt height.
- Icon buttons: use .frame(width: 44, height: 44) even if icon is 20-24pt.

VERTICAL RHYTHM:
- Consistent spacing within a section (all rows use same gap).
- Larger gap between sections than within sections.
- Example: 8pt between rows within a section, 24pt between sections.
- Headers get extra top spacing: 32pt above, 8pt below.

DENSITY GUIDELINES (from plan's design.density):
- "spacious": Use .large/.xLarge gaps. Generous padding (20-24pt). Best for meditation, health, reading apps.
- "standard": Use .medium gaps. 16pt padding. Default for most apps.
- "compact": Use .small/.xSmall gaps. 12pt padding. Best for data-heavy apps (finance, productivity).

CARD LAYOUT PATTERN:
```swift
VStack(alignment: .leading, spacing: AppTheme.Spacing.xSmall) {
    // Card content
}
.padding(AppTheme.Spacing.medium)
.background(AppTheme.Colors.surface)
.clipShape(RoundedRectangle(cornerRadius: AppTheme.Style.cornerRadius))
.shadow(color: .black.opacity(0.06), radius: 8, y: 4)
```

HORIZONTAL LAYOUTS:
- Space between icon and label: 8pt.
- Space between label and trailing value: use Spacer().
- Button bar with 2-3 buttons: HStack with spacing: 12pt.
- 4+ buttons: wrap in ScrollView(.horizontal).

SAFE AREA:
- Content respects safe areas by default — don't add redundant padding.
- Full-screen backgrounds use .ignoresSafeArea() to extend edge-to-edge.
- Overlays use .safeAreaInset(edge:) — never manual padding for safe areas.

# Speech

SPEECH RECOGNITION:
- import Speech; SFSpeechRecognizer()
- Requires NSSpeechRecognitionUsageDescription + NSMicrophoneUsageDescription (add CONFIG_CHANGES)
- SFSpeechRecognizer.requestAuthorization() for permission
- On-device: SFSpeechRecognizer(locale:), set requiresOnDeviceRecognition = true
- Live: SFSpeechAudioBufferRecognitionRequest + AVAudioEngine
- File: SFSpeechURLRecognitionRequest(url:)

# Storage Pattern Rules

## Default: In-Memory with Sample Data
Unless the user explicitly requests persistence (words like "save", "persist", "database", "storage", "SwiftData"), use **in-memory data with rich dummy data**:

| Data Type | Storage | API |
|-----------|---------|-----|
| App data (notes, tasks, items) | In-memory (DEFAULT) | Plain `struct`, `@Observable` arrays |
| Simple flags and settings (sort order, preferred units) | UserDefaults | `@AppStorage` |
| Transient UI state (sheet shown, selected tab) | In-memory | `@State` |

## In-Memory Models (Default)
Models are plain structs with a static `sampleData` array:

```swift
import Foundation

struct Note: Identifiable {
    let id: UUID
    var title: String
    var content: String
    var createdAt: Date
    var isPinned: Bool

    init(title: String, content: String = "", isPinned: Bool = false) {
        self.id = UUID()
        self.title = title
        self.content = content
        self.createdAt = Date()
        self.isPinned = isPinned
    }

    static let sampleData: [Note] = [
        Note(title: "Meeting Notes", content: "Discuss Q2 roadmap with the team", isPinned: true),
        Note(title: "Recipe: Pasta Carbonara", content: "Eggs, pecorino, guanciale, black pepper"),
        Note(title: "Book Recommendations", content: "The Midnight Library, Project Hail Mary"),
        Note(title: "Workout Plan", content: "Mon: Upper body, Wed: Legs, Fri: Cardio"),
        Note(title: "Gift Ideas", content: "Wireless earbuds, cookbook, plant pot"),
    ]
}
```

## In-Memory ViewModel (Default)
ViewModel holds data directly — no SwiftData, no modelContext:

```swift
import SwiftUI

@Observable
@MainActor
class NotesListViewModel {
    var notes: [Note] = Note.sampleData
    var searchText = ""

    var filteredNotes: [Note] {
        if searchText.isEmpty { return notes }
        return notes.filter { $0.title.localizedCaseInsensitiveContains(searchText) }
    }

    func addNote(title: String) {
        let note = Note(title: title)
        notes.insert(note, at: 0)
    }

    func deleteNote(_ note: Note) {
        notes.removeAll { $0.id == note.id }
    }
}
```

## App Entry Point (In-Memory — No Container)
```swift
@main
struct MyApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
```

## @AppStorage
Use for simple key-value settings:

```swift
@AppStorage("sortOrder") private var sortOrder = "date"
@AppStorage("showCompleted") private var showCompleted = true
```

## SwiftData (ONLY when user explicitly requests persistence)
Use `@Model`, `@Query`, `.modelContainer` ONLY if user says "save", "persist", "database", "storage", or "SwiftData":

```swift
import SwiftData

@Model
class Note {
    var title: String
    var content: String
    var createdAt: Date

    init(title: String, content: String = "") {
        self.title = title
        self.content = content
        self.createdAt = Date()
    }
}
```

# Swift Conventions

## Platform Requirements
- **Swift 6** language version
- **iOS 26+** deployment target
- **SwiftUI-first** architecture. UIKit is allowed only when no viable SwiftUI equivalent exists for a required feature.
- No Storyboards or XIBs

## Allowed Frameworks
Only import from this list:
- `SwiftUI`
- `Foundation`
- `SwiftData`
- `Observation`
- `OSLog` (for logging)
- `UIKit` (only for required bridges or APIs without a SwiftUI equivalent)
- `PhotosUI` (only when the app requires photo picking)
- `AVFoundation` (only when the app requires audio/video)
- `MapKit` (only when the app requires maps)
- `CoreLocation` (only when the app requires location)

## API Preferences
Use modern APIs — avoid deprecated alternatives:

| Use This | Not This |
|----------|----------|
| `foregroundStyle()` | `foregroundColor()` |
| `clipShape(.rect(cornerRadius:))` | `.cornerRadius()` |
| `@Observable` | `ObservableObject` |
| `@State` with `@Observable` | `@StateObject` |
| `.onChange(of:) { new in }` | `.onChange(of:) { old, new in }` (iOS 17+ form) |
| `.task { await … }` | `.onAppear { Task { await … } }` (use `.task` for async work) |
| `.onAppear { }` | *(acceptable for synchronous setup like passing modelContext)* |
| `NavigationStack` | `NavigationView` |

## Swift 6 Concurrency Safety

Swift 6 enables strict concurrency checking by default. All data races are compile-time errors.

### Sendable Conformance
- All types that cross isolation boundaries MUST conform to `Sendable`.
- Value types (structs, enums) with only Sendable stored properties are implicitly Sendable.
- Reference types (classes) must be `final` and have only immutable (`let`) Sendable stored properties, OR be marked `@unchecked Sendable` with manual thread safety (mutex/lock).
- Closures that cross isolation boundaries must be `@Sendable` — they cannot capture mutable local state.

### @MainActor Isolation
- ViewModels that update `@Published` or `@Observable` state MUST be annotated `@MainActor`.
- Services that read/write UI-bound state MUST be `@MainActor`.
- SwiftUI Views are implicitly `@MainActor`, so `@State var vm: SomeViewModel` works if `SomeViewModel` is `@MainActor`.
- Use `@MainActor` on the class/actor declaration, not on individual methods — partial isolation causes confusing errors.

### nonisolated Usage
- Computed properties that return Sendable values and don't access mutable actor state should be `nonisolated`.
- Protocol conformance requirements (e.g., `Hashable.hash(into:)`, `CustomStringConvertible.description`) on `@MainActor` types must be `nonisolated`.
- Use `nonisolated(unsafe)` only as a last resort for legacy API bridging.

### Crossing Isolation Boundaries
- Do not pass `@MainActor`-isolated values directly into APIs that execute in `@concurrent` contexts.
- Snapshot required values into local Sendable variables first, perform the concurrent call, then hop back to `MainActor` for state updates:
  ```swift
  // WRONG: passing actor-isolated self.items into concurrent context
  let result = await processor.process(items)

  // RIGHT: snapshot first
  let snapshot = items  // Copy Sendable data
  let result = await processor.process(snapshot)
  await MainActor.run { self.items = result }
  ```
- The `sending` parameter keyword marks closure parameters that cross isolation boundaries — use it when writing APIs that accept callbacks from different isolation domains.

### Global Variables
- Use `static let` for shared constants — `static var` is a mutable global and violates strict concurrency.
- If mutable global state is truly needed, wrap it in an `actor` or `@MainActor`-isolated type.
- `AppIntent` static properties (title, description) MUST be `static let`.

### Actor Re-entrancy
- Actor methods can suspend and resume — do not assume state is unchanged after an `await` inside an actor.
- Re-read state after suspension points; do not rely on pre-await values.

### @preconcurrency Import
- Use `@preconcurrency import FrameworkName` for frameworks not yet annotated for Sendable (e.g., some older Apple frameworks).
- This silences warnings for types the framework hasn't updated, without losing safety for your own code.

### Translation Framework
- For `.translationTask { session in ... }` flows, do NOT call `session.translate(...)` inside `@MainActor`-isolated methods.
- Perform translation work off the main actor and hop back to `@MainActor` only for UI state updates.

## Import Rules
- Start every file with the appropriate `import` statement
- Only import what's needed — don't import `SwiftUI` in a model file that only needs `Foundation`
- ViewModels may import `SwiftUI` and `UIKit` when needed for framework types or platform bridges, but must not declare Views/UIViewControllers

# Timers

TIMERS:
- Use TimelineView(.animation) for smooth countdown/stopwatch UI
- Timer.publish(every:on:in:).autoconnect() for periodic updates
- @State private var timeRemaining: TimeInterval for countdown state
- .onReceive(timer) { _ in } to update state each tick
- Format with Duration.formatted() or custom mm:ss formatter
- Invalidate timer in .onDisappear to prevent leaks

# Typography

IOS TYPE SCALE (use system text styles — NEVER .system(size:)):

| Style          | Size | Weight   | SwiftUI                   | Use Case                        |
|----------------|------|----------|---------------------------|---------------------------------|
| Large Title    | 34pt | Regular  | .largeTitle               | Screen titles (NavigationStack) |
| Title          | 28pt | Regular  | .title                    | Section headers                 |
| Title 2        | 22pt | Regular  | .title2                   | Sub-section headers             |
| Title 3        | 20pt | Regular  | .title3                   | Card titles                     |
| Headline       | 17pt | Semibold | .headline                 | Row titles, emphasized labels   |
| Body           | 17pt | Regular  | .body                     | Primary content text            |
| Callout        | 16pt | Regular  | .callout                  | Secondary descriptions          |
| Subheadline    | 15pt | Regular  | .subheadline              | Supporting text, timestamps     |
| Footnote       | 13pt | Regular  | .footnote                 | Tertiary info, disclaimers      |
| Caption        | 12pt | Regular  | .caption                  | Metadata, labels                |
| Caption 2      | 11pt | Regular  | .caption2                 | Smallest readable text          |

HIERARCHY RULES:
- ONE .largeTitle per screen (via .navigationTitle with .large display mode).
- .headline for list row titles and card headings.
- .body for primary content paragraphs and descriptions.
- .subheadline or .caption for metadata (dates, counts, secondary info).
- Visual hierarchy: at least 2 levels of contrast per screen (e.g. .headline + .caption).

FONT WEIGHT GUIDANCE:
- .regular: body text, descriptions.
- .medium: subtle emphasis, tab labels.
- .semibold: section headers, row titles, buttons.
- .bold: primary action labels, important callouts.
- AVOID .ultraLight, .thin, .light — poor readability, especially on small screens.

FONT DESIGN:
- .fontDesign(.rounded): friendly, playful apps (fitness, kids, social).
- .fontDesign(.serif): editorial, premium (news, reading, luxury).
- .fontDesign(.monospaced): technical, developer tools, code display.
- .fontDesign(.default): neutral, professional (productivity, finance).
- Apply on outermost container — it cascades to all children.

DYNAMIC TYPE:
- System text styles automatically scale with user's accessibility settings.
- NEVER use .font(.system(size: N)) — it opts out of Dynamic Type.
- If a layout breaks at larger type sizes, use .minimumScaleFactor(0.8) as last resort.
- Test with largest accessibility size: Xcode → Environment Overrides → Dynamic Type.

LINE SPACING & READABILITY:
- System styles include appropriate leading — don't override unless necessary.
- For long-form text, .lineSpacing(4) improves readability.
- Maximum comfortable line length: ~70 characters. Use .frame(maxWidth: 600) for wide screens.

NUMBER & DATA DISPLAY:
- Use .monospacedDigit() for numbers that change (timers, counters, prices).
- This prevents layout jitter as digits change width.
- Example: Text(price).font(.title2).monospacedDigit()

# View Composition Rules

## @ViewBuilder Computed Properties
Every meaningful section of a view should be a `@ViewBuilder` computed property:

```swift
struct TaskListView: View {
    var body: some View {
        NavigationStack {
            VStack {
                headerView
                taskList
                addButton
            }
        }
    }

    private var headerView: some View {
        Text("My Tasks")
            .font(.largeTitle)
            .padding()
    }

    private var taskList: some View {
        List(tasks) { task in
            TaskRow(task: task)
        }
    }

    private var addButton: some View {
        Button("Add Task") {
            showAddSheet = true
        }
        .buttonStyle(.borderedProminent)
    }
}
```

## Size Guidelines
- Each computed property: **15-40 lines**
- If a section exceeds 40 lines, extract it to a separate struct
- If a section is reused in multiple views, extract to a separate file

## Section Naming Pattern
Use descriptive suffixes:
- `headerSection`, `headerView` — top area
- `contentSection`, `contentView` — main content
- `footerSection` — bottom area
- `{feature}Section` — feature-specific (e.g., `profileSection`, `statsSection`)
- `{action}Button` — action buttons (e.g., `addButton`, `saveButton`)

## When to Extract to Separate Struct
Extract a section into its own `View` struct when:
1. It exceeds **40 lines** of code
2. It's **reused** across multiple views
3. It has its own **state** (needs `@State` or `@Binding`)
4. It represents a **distinct UI component** (e.g., a card, a row)

## Body Should Only Contain Property References
The `body` property should be a composition of computed properties — not raw implementation:

```swift
// Good — body is a clear outline
var body: some View {
    ScrollView {
        headerSection
        featureCards
        recentActivity
    }
}

// Bad — body contains raw implementation
var body: some View {
    ScrollView {
        VStack(alignment: .leading) {
            Text("Welcome")
                .font(.largeTitle)
            // ... 50 more lines
        }
    }
}
```

# View Complexity

VIEW BODY COMPLEXITY (CRITICAL):

- If a View `body` grows beyond about 30 lines, extract sections into computed properties.
- If nesting is deeper than 3 levels, flatten the structure by extracting sub-sections.
- Prefer a body that reads like a table of contents:
  - `headerSection`
  - `contentSection`
  - `footerSection`

REQUIRED PATTERN:

```swift
var body: some View {
    VStack(spacing: AppTheme.Spacing.medium) {
        headerSection
        contentSection
        actionsSection
    }
}

private var headerSection: some View { ... }
private var contentSection: some View { ... }
private var actionsSection: some View { ... }
```

TYPE-CHECK TIMEOUT FIX:

- Error pattern: "The compiler is unable to type-check this expression in reasonable time"
- Fix by splitting large expressions:
  - Move long `HStack/VStack/ZStack` branches into computed properties
  - Move complex `Chart` blocks into computed properties
  - Move long modifier chains into intermediate variables/properties
- Keep behavior exactly the same; only refactor structure.

WHEN EDITING EXISTING FILES:

- If your new change pushes body size over the threshold, include the refactor in the same edit.
- Do not leave a giant body as technical debt.

# Website Links

WEBSITE LINKS:
- Link("Visit Website", destination: URL(string: "https://...")!) for simple links
- Or @Environment(\.openURL) var openURL; Button { openURL(url) } for custom styling
- No permissions needed for external URLs
- Safari opens by default; use SFSafariViewController via UIViewControllerRepresentable for in-app browser
- Validate URL before force-unwrapping: guard let url = URL(string:) else { return }

# Widgets

WIDGETS (WidgetKit):
FRAMEWORK: import WidgetKit

SETUP:
- Requires separate widget extension target (kind: "widget" in plan extensions array)
- Use AppGroup for data sharing between app and widget (auto-configured in project.yml)
- Widget code goes in Targets/{WidgetName}/ directory

TIMELINE ENTRY (data for a single widget render):
struct MyEntry: TimelineEntry {
    let date: Date
    let title: String
    let value: Double
}

TIMELINE PROVIDER (supplies entries to the system):
struct MyProvider: TimelineProvider {
    func placeholder(in context: Context) -> MyEntry {
        MyEntry(date: .now, title: "Placeholder", value: 0)
    }
    func getSnapshot(in context: Context, completion: @escaping (MyEntry) -> Void) {
        completion(MyEntry(date: .now, title: "Snapshot", value: 42))
    }
    func getTimeline(in context: Context, completion: @escaping (Timeline<MyEntry>) -> Void) {
        let entry = MyEntry(date: .now, title: "Current", value: 42)
        let timeline = Timeline(entries: [entry], policy: .after(.now.addingTimeInterval(3600)))
        completion(timeline)
    }
}

WIDGET VIEW:
struct MyWidgetView: View {
    var entry: MyProvider.Entry
    @Environment(\.widgetFamily) var family
    var body: some View {
        switch family {
        case .systemSmall:
            VStack { Text(entry.title).font(.headline); Text("\(Int(entry.value))").font(.largeTitle) }
        case .systemMedium:
            HStack { VStack(alignment: .leading) { Text(entry.title); Text("\(Int(entry.value))").font(.title) }; Spacer() }
        default:
            Text(entry.title)
        }
    }
}

WIDGET DEFINITION:
struct MyWidget: Widget {
    let kind: String = "MyWidget"
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: MyProvider()) { entry in
            MyWidgetView(entry: entry)
                .containerBackground(.fill.tertiary, for: .widget)  // REQUIRED iOS 17+
        }
        .configurationDisplayName("My Widget")
        .description("Shows current status")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}

WIDGET BUNDLE (when multiple widgets exist):
@main
struct MyWidgetBundle: WidgetBundle {
    var body: some Widget {
        MyWidget()
        AnotherWidget()
    }
}

MANDATORY FILES (every widget extension MUST have ALL of these):
1. {Name}Bundle.swift — @main WidgetBundle entry point. Without this, the extension has no entry point → linker error "undefined symbol: _main" → CodeSign failure.
2. Provider.swift — TimelineProvider implementation.
3. WidgetView.swift — The widget's SwiftUI view.
4. (Optional) Intent.swift — AppIntent for interactive widgets (tap-to-complete, etc.).

CRITICAL RULES:
- .containerBackground(.fill.tertiary, for: .widget) is REQUIRED on the widget view in iOS 17+. Without it, the widget renders with no background.
- The @main entry point is MANDATORY. Use @main on WidgetBundle (multiple widgets) or Widget (single widget). An extension target with NO @main will fail to link.
- Shared data types between app and widget go in the Shared/ directory at the project root (both targets compile it). NEVER define shared types only in the main app's Models/ — the widget extension cannot see them.
- Widget views must be self-contained — they cannot use @StateObject, @ObservedObject, or network calls. All data comes through the TimelineEntry.
- Use .supportedFamilies() to declare which sizes the widget supports.
- AppIntent static properties MUST use "static let" (not "static var") for Swift 6 concurrency safety. Using "static var" causes "not concurrency-safe because it is nonisolated global shared mutable state" error.

