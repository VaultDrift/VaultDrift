# VaultDrift Mobile

React Native mobile app for VaultDrift - Secure Cloud Storage on iOS and Android.

## Features

- **Authentication**: Secure login with JWT tokens and biometric authentication (Face ID / Fingerprint)
- **File Management**: Browse, upload, download, and share files
- **Camera Upload**: Direct photo capture and upload
- **Offline Support**: View cached files when offline
- **Auto-Upload**: Automatically upload new photos
- **Biometric Security**: Unlock app with Face ID or Fingerprint
- **Dark Mode**: Full dark theme support

## Tech Stack

- React Native 0.76 + Expo SDK 52
- TypeScript
- React Navigation 7
- TanStack Query (React Query)
- Zustand (State Management)
- Axios (API Client)

## Getting Started

### Prerequisites

- Node.js 18+
- npm or yarn
- iOS: macOS with Xcode
- Android: Android Studio with SDK

### Installation

```bash
# Install dependencies
npm install

# Install Expo CLI globally (if not already installed)
npm install -g expo-cli

# Start the development server
npx expo start
```

### Running on Devices

**iOS Simulator:**
```bash
npx expo start --ios
```

**Android Emulator:**
```bash
npx expo start --android
```

**Physical Device:**
Scan the QR code in Expo Go app or use `expo-dev-client` for development builds.

### Building Production Apps

**iOS:**
```bash
eas build --platform ios
```

**Android:**
```bash
eas build --platform android
```

## Project Structure

```
mobile/
├── App.tsx                    # App entry point
├── app.json                   # Expo configuration
├── package.json               # Dependencies
├── src/
│   ├── api/
│   │   └── client.ts          # Axios API client
│   ├── navigation/
│   │   ├── RootNavigator.tsx  # Root navigation
│   │   ├── AuthNavigator.tsx  # Auth flow navigation
│   │   └── MainNavigator.tsx  # Main app navigation (tabs)
│   ├── screens/
│   │   ├── auth/
│   │   │   ├── LoginScreen.tsx      # Login
│   │   │   └── ServerSetupScreen.tsx # Server configuration
│   │   ├── main/
│   │   │   ├── FilesScreen.tsx      # File browser
│   │   │   ├── SharedScreen.tsx     # Shared files
│   │   │   ├── UploadsScreen.tsx    # Upload manager
│   │   │   └── SettingsScreen.tsx   # Settings
│   │   └── LoadingScreen.tsx        # Loading screen
│   ├── stores/
│   │   ├── authStore.ts       # Authentication state
│   │   └── settingsStore.ts   # Settings state
│   └── utils/
│       └── biometric.ts       # Biometric auth utilities
└── assets/                    # Images, icons, splash
```

## Features Detail

### Authentication
- JWT token storage in SecureStore
- Automatic token refresh
- Biometric authentication (Face ID / Fingerprint)
- Secure token revocation on logout

### File Management
- Browse files and folders
- Upload from device storage
- Upload from camera
- Share files via share links
- Download for offline access

### Settings
- Dark/Light theme toggle
- Auto-upload photos
- Wi-Fi only upload option
- Push notifications
- Biometric unlock

## Configuration

### Server URL
Set your VaultDrift server URL in the app during first login. The app supports:
- Local network IPs
- HTTPS domains
- Custom ports

### Environment Variables
Create a `.env` file for environment-specific configuration:

```
API_URL=https://your-server.com
```

## Security

- All tokens stored in platform secure storage
- Biometric authentication available
- Certificate pinning supported
- No sensitive data in plain storage

## License

MIT
