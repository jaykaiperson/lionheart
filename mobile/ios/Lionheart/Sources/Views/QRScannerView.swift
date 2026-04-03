import SwiftUI
import AVFoundation

struct QRScannerView: View {
    let onScan: (String) -> Void
    @Environment(\.dismiss) var dismiss
    @State private var permissionStatus: AVAuthorizationStatus = .notDetermined
    
    var body: some View {
        ZStack {
            if permissionStatus == .authorized {
                #if targetEnvironment(simulator)
                VStack {
                    Image(systemName: "camera.slash.fill").font(.largeTitle).padding()
                    Text("no_camera_desc").multilineTextAlignment(.center).padding()
                    Button("close") { dismiss() }.buttonStyle(.bordered)
                }
                #else
                ZStack {
                    QRScannerViewControllerRepresentable { code in
                        onScan(code)
                        dismiss()
                    }
                    .ignoresSafeArea()
                    
                    Color.black.opacity(0.6)
                        .mask(
                            ZStack {
                                Color.white
                                RoundedRectangle(cornerRadius: 24)
                                    .frame(width: 260, height: 260)
                                    .blendMode(.destinationOut)
                            }
                        )
                        .ignoresSafeArea()
                    
                    RoundedRectangle(cornerRadius: 24)
                        .stroke(Color.white.opacity(0.5), lineWidth: 3)
                        .frame(width: 260, height: 260)
                    
                    VStack {
                        HStack {
                            Spacer()
                            Button(action: { dismiss() }) {
                                Image(systemName: "xmark.circle.fill")
                                    .font(.system(size: 32))
                                    .foregroundColor(.white.opacity(0.8))
                                    .padding()
                            }
                        }
                        Spacer()
                        Text("scan_qr_title")
                            .font(.headline)
                            .foregroundColor(.white)
                            .padding(.bottom, 60)
                    }
                }
                #endif
            } else if permissionStatus == .denied || permissionStatus == .restricted {
                VStack(spacing: 16) {
                    Image(systemName: "camera.slash").font(.system(size: 40)).foregroundColor(.red)
                    Text("camera_permission_needed").font(.headline).multilineTextAlignment(.center)
                    Button("grant_access") {
                        if let url = URL(string: UIApplication.openSettingsURLString) {
                            UIApplication.shared.open(url)
                        }
                    }.buttonStyle(.borderedProminent)
                    Button("cancel") { dismiss() }.padding(.top)
                }
                .padding()
            } else {
                Color.black.ignoresSafeArea()
            }
        }
        .onAppear(perform: checkCamera)
    }
    
    private func checkCamera() {
        let status = AVCaptureDevice.authorizationStatus(for: .video)
        if status == .notDetermined {
            AVCaptureDevice.requestAccess(for: .video) { granted in
                DispatchQueue.main.async {
                    self.permissionStatus = granted ? .authorized : .denied
                }
            }
        } else {
            self.permissionStatus = status
        }
    }
}

#if !targetEnvironment(simulator)
struct QRScannerViewControllerRepresentable: UIViewControllerRepresentable {
    let onDetect: (String) -> Void
    func makeUIViewController(context: Context) -> ScannerViewController {
        let vc = ScannerViewController()
        vc.onDetect = onDetect
        return vc
    }
    func updateUIViewController(_ uiViewController: ScannerViewController, context: Context) {}
}

class ScannerViewController: UIViewController, AVCaptureMetadataOutputObjectsDelegate {
    var captureSession: AVCaptureSession!
    var previewLayer: AVCaptureVideoPreviewLayer!
    var onDetect: ((String) -> Void)?
    private var isScanned = false
    
    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = UIColor.black
        captureSession = AVCaptureSession()
        guard let videoCaptureDevice = AVCaptureDevice.default(for: .video) else { return }
        guard let videoInput = try? AVCaptureDeviceInput(device: videoCaptureDevice) else { return }
        if (captureSession.canAddInput(videoInput)) { captureSession.addInput(videoInput) } else { return }
        let metadataOutput = AVCaptureMetadataOutput()
        if (captureSession.canAddOutput(metadataOutput)) {
            captureSession.addOutput(metadataOutput)
            metadataOutput.setMetadataObjectsDelegate(self, queue: DispatchQueue.main)
            metadataOutput.metadataObjectTypes = [.qr]
        } else { return }
        previewLayer = AVCaptureVideoPreviewLayer(session: captureSession)
        previewLayer.frame = view.layer.bounds
        previewLayer.videoGravity = .resizeAspectFill
        view.layer.addSublayer(previewLayer)
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in self?.captureSession.startRunning() }
    }
    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        if captureSession?.isRunning == false { DispatchQueue.global(qos: .userInitiated).async { [weak self] in self?.captureSession.startRunning() } }
    }
    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        if captureSession?.isRunning == true { DispatchQueue.global(qos: .userInitiated).async { [weak self] in self?.captureSession.stopRunning() } }
    }
    func metadataOutput(_ output: AVCaptureMetadataOutput, didOutput metadataObjects: [AVMetadataObject], from connection: AVCaptureConnection) {
        if isScanned { return }
        if let metadataObject = metadataObjects.first {
            guard let readableObject = metadataObject as? AVMetadataMachineReadableCodeObject else { return }
            guard let stringValue = readableObject.stringValue else { return }
            isScanned = true
            AudioServicesPlaySystemSound(SystemSoundID(kSystemSoundID_Vibrate))
            onDetect?(stringValue)
        }
    }
}
#endif
