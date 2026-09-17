package main

import (
"fmt"
"net/http"
"sync"

"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

type Message struct {
Type      string `json:"type"`
Group     string `json:"group"`
User      string `json:"user"`
Text      string `json:"text"`
Target    string `json:"target"`
File      string `json:"data"`
FileType  string `json:"fileType"`
Video     bool   `json:"video"`
Offer     any    `json:"offer"`
Answer    any    `json:"answer"`
Candidate any    `json:"candidate"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
return
}
defer conn.Close()

mu.Lock()
clients[conn] = true
mu.Unlock()

for {
var msg Message
err := conn.ReadJSON(&msg)
if err != nil {
mu.Lock()
delete(clients, conn)
mu.Unlock()
break
}

mu.Lock()
for clientConn := range clients {
clientConn.WriteJSON(msg)
}
mu.Unlock()
}
}

var htmlPage = `<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>هوت واتساب - Hot WhatsApp</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, sans-serif; background: #111b21; color: white; display: flex; height: 100vh; margin: 0; overflow: hidden; }
        .sidebar { width: 300px; background: #202c33; border-left: 1px solid #2a3942; display: flex; flex-direction: column; }
        .sidebar-header { padding: 15px; background: #202c33; border-bottom: 1px solid #2a3942; display: flex; align-items: center; justify-content: space-between; }
        .logo-area { display: flex; align-items: center; gap: 10px; }
        .logo-area img { width: 35px; height: 35px; border-radius: 50%; object-fit: cover; border: 2px solid #00a884; }
        .group-list { flex: 1; overflow-y: auto; }
        .group-item { padding: 15px; border-bottom: 1px solid #2a3942; cursor: pointer; display: flex; align-items: center; gap: 10px; }
        .group-item:hover { background: #2a3942; }
        .avatar { width: 40px; height: 40px; background: #00a884; border-radius: 50%; display: flex; justify-content: center; align-items: center; font-weight: bold; }
        
        .main-chat { flex: 1; display: flex; flex-direction: column; background: #0b141a; position: relative; }
        .header { background: #202c33; padding: 15px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #2a3942; }
        .chat-box { flex: 1; padding: 20px; overflow-y: auto; display: flex; flex-direction: column; gap: 10px; }
        .bubble { padding: 10px 15px; border-radius: 10px; max-width: 65%; font-size: 15px; word-wrap: break-word; }
        .msg-in { background: #202c33; align-self: flex-start; }
        .msg-out { background: #005c4b; align-self: flex-end; }
        .bubble small { display: block; font-size: 11px; color: #aebac1; margin-bottom: 5px; font-weight: bold; }
        
        .input-bar { display: flex; padding: 15px; background: #202c33; gap: 10px; align-items: center; }
        .input-bar input[type="text"] { flex: 1; padding: 12px; border-radius: 20px; border: none; background: #2a3942; color: white; outline: none; }
        button { background: #00a884; color: white; border: none; padding: 10px 15px; border-radius: 8px; cursor: pointer; font-weight: bold; }

        #call-modal { display: none; position: absolute; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.85); z-index: 100; justify-content: center; align-items: center; }
        .modal-box { background: #202c33; padding: 30px; border-radius: 15px; text-align: center; width: 300px; border: 2px solid #00a884; }
        .btn-accept { background: #00a884; padding: 10px 20px; margin: 10px; border-radius: 50px; }
        .btn-reject { background: #f15c6d; padding: 10px 20px; margin: 10px; border-radius: 50px; }

        #call-screen { display: none; position: absolute; top:0; left:0; width:100%; height:100%; background:#111b21; z-index:50; flex-direction:column; }
        .video-container { flex:1; display:flex; justify-content:center; align-items:center; gap:20px; padding:20px; }
        video { background: #000; border-radius: 15px; border: 2px solid #2a3942; width: 45%; max-width: 500px; aspect-ratio: 16/9; object-fit: cover; }
        .controls-bar { display: flex; justify-content: center; gap: 20px; padding: 20px; background: #202c33; }
    </style>
</head>
<body>

<div id="call-modal">
    <div class="modal-box">
        <h2 style="margin:0; color:#00a884;">مكالمة واردة 📞</h2>
        <p>من: <strong id="caller-name"></strong></p>
        <button class="btn-accept" onclick="acceptCall()">رد ✅</button>
        <button class="btn-reject" onclick="rejectCall()">رفض ❌</button>
    </div>
</div>

<div class="sidebar">
    <div class="sidebar-header">
        <div class="logo-area">
            <img src="https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=100&auto=format&fit=crop&q=80" alt="Logo">
            <strong style="color:#00a884; font-size: 15px;">هوت واتساب</strong>
        </div>
        <button onclick="createGroup()">+ مجموعة</button>
    </div>
    <div class="group-list" id="group-list"></div>
</div>

<div class="main-chat">
    <div class="header">
        <strong id="group-title">الجروب العام</strong>
        <div id="call-buttons" style="display:flex; gap:10px;">
            <button onclick="startCall(false)">📞 صوت</button>
            <button onclick="startCall(true)">📹 فيديو</button>
        </div>
    </div>

    <div class="chat-box" id="messages"></div>

    <div class="input-bar">
        <label style="cursor:pointer; font-size:22px;">📎 
            <input type="file" style="display:none;" accept="image/*,video/*" onchange="sendFile(this)">
        </label>
        <input type="text" id="msgInput" placeholder="اكتب رسالتك..." onkeypress="if(event.key==='Enter') sendText()">
        <button onclick="sendText()">إرسال ➤</button>
    </div>

    <div id="call-screen">
        <div class="video-container">
            <video id="localVideo" autoplay playsinline muted></video>
            <video id="remoteVideo" autoplay playsinline></video>
        </div>
        <div class="controls-bar">
            <button id="btn-toggle-cam" style="background:#00a884; border-radius:50px; padding:12px 25px;" onclick="toggleCamera()">إيقاف الكاميرا 🚫</button>
            <button style="background:#f15c6d; border-radius:50px; padding:12px 25px;" onclick="endCall(true)">إنهاء المكالمة ❌</button>
        </div>
    </div>
</div>

<script>
    if (window.Notification && Notification.permission !== "granted") {
        Notification.requestPermission();
    }

    function showNotification(sender, text) {
        if (window.Notification && Notification.permission === "granted") {
            new Notification("رسالة جديدة من: " + sender, {
                body: text,
                icon: "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=100&auto=format&fit=crop&q=80"
            });
        }
    }

    const myName = prompt("أهلاً بك في هوت واتساب، اكتب اسمك:") || "مستخدم_" + Math.floor(Math.random()*100);
    let currentGroup = "الجروب العام";
    addGroupToList("الجروب العام");

    const protocol = window.location.protocol === "https:" ? "wss://" : "ws://";
    const ws = new WebSocket(protocol + window.location.host + "/ws");

    const rtcConfig = { iceServers: [{ urls: "stun:stun.l.google.com:19302" }] };
    let pc, localStream, pendingCaller = null, pendingCallVideo = false, currentPeerName = null, isVideoMuted = false;

    ws.onmessage = async (event) => {
        const data = JSON.parse(event.data);

        if (data.type === "chat" && data.group === currentGroup) {
            appendMessage(data.user, data.text.replace(/</g, "&lt;"), data.user === myName, false);
            if (data.user !== myName) {
                showNotification(data.user, data.text);
            }
        } else if (data.type === "file" && data.group === currentGroup) {
            let content = data.fileType === 'video' 
                ? '<video src="' + data.data + '" controls style="max-width:100%; border-radius:8px;"></video>'
                : '<img src="' + data.data + '" style="max-width:100%; border-radius:8px;">';
            appendMessage(data.user, content, data.user === myName, true);
        } else if (data.type === "ring" && data.target === myName) {
            pendingCaller = data.user;
            pendingCallVideo = data.video;
            document.getElementById("caller-name").innerText = data.user + (data.video ? " (فيديو)" : " (صوت)");
            document.getElementById("call-modal").style.display = "flex";
            showNotification(data.user, "مكالمة واردة..");
        } else if (data.type === "accept-call" && data.target === myName) {
            setupPeerConnection(data.user);
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);
            ws.send(JSON.stringify({ type: "offer", target: data.user, user: myName, offer: offer }));
        } else if (data.type === "reject-call" && data.target === myName) {
            alert(data.user + " رفض المكالمة.");
            endCallLocal();
        } else if (data.type === "end-call" && data.target === myName) {
            endCallLocal();
        }

        if (data.target !== myName) return;
        if (data.type === "offer") {
            setupPeerConnection(data.user);
            await pc.setRemoteDescription(new RTCSessionDescription(data.offer));
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            ws.send(JSON.stringify({ type: "answer", target: data.user, user: myName, answer: answer }));
        } else if (data.type === "answer") {
            await pc.setRemoteDescription(new RTCSessionDescription(data.answer));
        } else if (data.type === "ice-candidate") {
            await pc.addIceCandidate(new RTCIceCandidate(data.candidate));
        }
    };

    function createGroup() {
        const name = prompt("اسم المجموعة:");
        if (name) addGroupToList(name);
    }
    function addGroupToList(name) {
        document.getElementById("group-list").innerHTML += '<div class="group-item" onclick="openGroup(\'' + name + '\')"><div class="avatar">' + name.charAt(0) + '</div><strong>' + name + '</strong></div>';
    }
    function openGroup(name) {
        currentGroup = name;
        document.getElementById("group-title").innerText = name;
        document.getElementById("messages").innerHTML = '<div style="text-align:center; color:#00a884;">المجموعة: ' + name + '</div>';
    }

    function sendText() {
        const input = document.getElementById("msgInput");
        if (input.value.trim()) {
            ws.send(JSON.stringify({ type: "chat", group: currentGroup, user: myName, text: input.value.trim() }));
            input.value = "";
        }
    }

    function sendFile(input) {
        if (input.files[0]) {
            const file = input.files[0];
            const reader = new FileReader();
            reader.onload = e => {
                const type = file.type.startsWith('video') ? 'video' : 'image';
                ws.send(JSON.stringify({ type: "file", fileType: type, group: currentGroup, user: myName, data: e.target.result }));
            };
            reader.readAsDataURL(file);
        }
    }

    function appendMessage(user, content, isMe, isFile) {
        const div = document.createElement("div");
        div.className = "bubble " + (isMe ? "msg-out" : "msg-in");
        div.innerHTML = '<small>' + user + '</small>' + content;
        const box = document.getElementById("messages");
        box.appendChild(div);
        box.scrollTop = box.scrollHeight;
    }

    async function startCall(videoEnabled) {
        const target = prompt("اسم الشخص للاتصال به:");
        if (!target) return;
        currentPeerName = target;
        await initMedia(videoEnabled);
        document.getElementById("call-screen").style.display = "flex";
        ws.send(JSON.stringify({ type: "ring", target: target, user: myName, video: videoEnabled }));
    }

    async function acceptCall() {
        document.getElementById("call-modal").style.display = "none";
        currentPeerName = pendingCaller;
        await initMedia(pendingCallVideo);
        document.getElementById("call-screen").style.display = "flex";
        ws.send(JSON.stringify({ type: "accept-call", target: pendingCaller, user: myName }));
    }

    function rejectCall() {
        document.getElementById("call-modal").style.display = "none";
        ws.send(JSON.stringify({ type: "reject-call", target: pendingCaller, user: myName }));
    }

    async function initMedia(video) {
        if (!localStream) {
            localStream = await navigator.mediaDevices.getUserMedia({ video: video, audio: true });
            document.getElementById("localVideo").srcObject = localStream;
            const camBtn = document.getElementById("btn-toggle-cam");
            if (!video) {
                camBtn.innerText = "صوت فقط 📞";
                camBtn.disabled = true;
            } else {
                camBtn.innerText = "إيقاف الكاميرا 🚫";
                camBtn.disabled = false;
                isVideoMuted = false;
            }
        }
    }

    function setupPeerConnection(targetUser) {
        pc = new RTCPeerConnection(rtcConfig);
        localStream.getTracks().forEach(track => pc.addTrack(track, localStream));
        pc.ontrack = e => document.getElementById("remoteVideo").srcObject = e.streams[0];
        pc.onicecandidate = e => {
            if (e.candidate) ws.send(JSON.stringify({ type: "ice-candidate", target: targetUser, user: myName, candidate: e.candidate }));
        };
    }

    function toggleCamera() {
        if (localStream && localStream.getVideoTracks().length > 0) {
            isVideoMuted = !isVideoMuted;
            localStream.getVideoTracks()[0].enabled = !isVideoMuted;
            const btn = document.getElementById("btn-toggle-cam");
            btn.innerText = isVideoMuted ? "تشغيل الكاميرا 📹" : "إيقاف الكاميرا 🚫";
        }
    }

    function endCall(sendSignal = false) {
        if (sendSignal && currentPeerName) ws.send(JSON.stringify({ type: "end-call", target: currentPeerName, user: myName }));
        endCallLocal();
    }

    function endCallLocal() {
        if (localStream) { localStream.getTracks().forEach(t => t.stop()); localStream = null; }
        if (pc) { pc.close(); pc = null; }
        document.getElementById("call-screen").style.display = "none";
        document.getElementById("remoteVideo").srcObject = null;
        currentPeerName = null;
    }
</script>
</body>
</html>`

func main() {
http.HandleFunc("/ws", handleConnections)
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Write([]byte(htmlPage))
})

fmt.Println("🚀 سيرفر هوت واتساب يعمل الآن في ملف واحد على المنفذ 8080...")
http.ListenAndServe(":8080", nil)
}
