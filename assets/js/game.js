window.addEventListener("DOMContentLoaded", () => {

    const startButton = document.getElementById("startButton")
    const playerList = document.getElementById("playerList")
    const roleContainer = document.getElementById('roleContainer');
    const roleText = document.getElementById('roleText');
    const gameArea = document.getElementById('gameArea');
    const sendButton = document.getElementById("sendButton")
    const microphoneButton = document.getElementById("micButton")
    const cameraButton = document.getElementById("cameraButton")

    const peerConnections = {}

    const configuration = {
        iceServers: [
            { urls: 'stun:stun.l.google.com:19302' },
            { urls: 'stun:stun1.l.google.com:19302' }
        ]
    }

    cameraButton.disabled = true;
    microphoneButton.disabled = true;
    
    var roomCode = window.location.pathname.split("/").pop()
    console.log("roomCode after declaring the variable:", roomCode)
    var ws
    var localStream
    var currentUserName = getCurrentUserNameFromURL()

    startButton.addEventListener('click', onStartButtonClick);
    sendButton.addEventListener('click', onSendButtonClick);
    microphoneButton.addEventListener('click', toggleMicrophone);
    cameraButton.addEventListener('click', toggleCamera);
    document.getElementById("messageInput").addEventListener("keypress", (e) => {
        if (e.key === "Enter") onSendButtonClick();
    });

    const observer = new MutationObserver(checkPlayerCount)
    observer.observe(playerList, { childList: true })
    checkPlayerCount()

    function onStartButtonClick() {
        fetch("/start", {
            method: "POST",
            headers: { 'Content-Type': "application/json" },
            body: JSON.stringify({ roomCode, currentUserName })
        })
        .then(res => res.ok ? res.json() : Promise.reject("Failed to start game"))
        .then(data => {
            const me = data.find(p => p.name === currentUserName);
            if (me) startGameUI(me);
        })
        .catch(err => console.error(err));
    }

    function onSendButtonClick() {
        console.log("Send button")
        const input = document.getElementById("messageInput");
        if (input.value.trim() && ws) {
            const messageData = {
                sender: currentUserName,
                content: input.value.trim(),
                timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
                type: "text"
            }
            console.log("SENDING (text)")
            console.log("message Data:", messageData)
            ws.send(JSON.stringify(messageData))
            input.value = ""
            console.log("MESSAGE (text) has been sent")
        }
    }

    function startGameUI(me) {
        document.getElementById("gameInfo").style.display = "none"
        document.getElementById("statusBar").style.display = "none"
        document.getElementsByTagName("header")[0].style.display = "none"
        startButton.style.display = "none"

        roleText.textContent = `Your role: ${me.role}`
        roleContainer.style.display = "flex"

        setTimeout(async () => {
            roleContainer.style.display = "none"
            gameArea.style.display = "grid"

            const mediaReady = await setupLocalMedia()
            
            if (mediaReady) {
                cameraButton.disabled = false
                microphoneButton.disabled = false

                connectToSignalingServer();
            }

        }, 3000)
    }

    async function setupLocalMedia() {
        try {
            localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
            console.log("Audio tracks:", localStream.getAudioTracks())

            const videoContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`)
            if (videoContainer) {
                const video = videoContainer.querySelector("video.video-element")
                const placeholder = videoContainer.querySelector(".video-placeholder")

                placeholder.style.display = 'none'
                video.srcObject = localStream
                video.muted = true

                video.style.display = "block" // or "flex", "inline-block", etc. depending on the css(check it up later)
            }

            return true
        } catch (err) {
            console.error("Error accessing media devices:", err)
            alert("Could not access your camera and microphone. Please check permissions")
            return false
        }
    }

    function connectToSignalingServer() {
        const protocol = window.location.protocol === "https:" ? "wss" : "ws"
        const host = window.location.host
        ws = new WebSocket(`${protocol}://${host}/ws/chat?room=${roomCode}&user=${currentUserName}`)
    
        ws.onopen = function() {
            console.log("Connected to WebSocket server")
        }
    
        ws.onmessage = async function(event) {
            console.log("onmessage")
            console.log("⬇️ RECEIVED (raw):", event.data)
            const message = JSON.parse(event.data)

            switch (message.type) {
                case "text":
                    console.log("Received text")
                    displayChatMessage(message)
                    break
                case "player-joined":
                    console.log(`${message.name} joined. Creating WebRTC offer...`)
                    createAndSendOffer(message.name)
                    break
                case "offer":
                    console.log(`Received offer from ${message.sender}.`)
                    await handleOffer(message.sender, message.sdp)
                    break
                case "answer":
                    console.log(`Received answer from ${message.sender}.`)
                    await handleAnswer(message.sender, message.sdp)
                    break
                case "candidate":
                    console.log("Received candidate")
                    await handleIceCandidate(message.sender, message.candidate)
                    break;
            }
        }
    
        ws.onclose = function() {
            console.log("WebSocket connection closed, retrying...")
            setTimeout(connectToSignalingServer, 1000)
        }
    
        ws.onerror = function(error) {
            console.error("WebSocket error:", error)
        }
    }

    function createPeerConnection(playerName) {
        if (peerConnections[playerName]) return peerConnections[playerName]

        const pc = new RTCPeerConnection(configuration)
        peerConnections[playerName] = pc

        localStream.getTracks().forEach(track => pc.addTrack(track, localStream))

        pc.ontrack = (event) => {
            const remoteVideoContainer = document.querySelector(`.video-container[data-player-name="${playerName}"]`)
            if (remoteVideoContainer) {
                const video = remoteVideoContainer.querySelector("video.video-element")
                remoteVideoContainer.querySelector('.video-placeholder').style.display = 'none'
                video.srcObject = event.streams[0]
            }
        }

        pc.onicecandidate = (event) => {
            if (event.candidate) {
                ws.send(JSON.stringify({
                    type: 'candidate',
                    sender: currentUserName,
                    receiver: playerName,
                    candidate: event.candidate,
                }))
            }
        }

        return pc
    }

    async function createAndSendOffer(playerName) {
        const pc = createPeerConnection(playerName);
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        
        ws.send(JSON.stringify({
            type: 'offer',
            sender: currentUserName,
            receiver: playerName,
            sdp: pc.localDescription,
        }));
    }

    async function handleOffer(senderName, sdp) {
        const pc = createPeerConnection(senderName);
        await pc.setRemoteDescription(new RTCSessionDescription(sdp));
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        
        ws.send(JSON.stringify({
            type: 'answer',
            sender: currentUserName,
            receiver: senderName,
            sdp: pc.localDescription,
        }));
    }

    async function handleAnswer(senderName, sdp) {
        const pc = peerConnections[senderName]
        if (pc) {
            await pc.setRemoteDescription(new RTCSessionDescription(sdp))
        }
    }

    async function handleIceCandidate(senderName, candidate) {
        const pc = peerConnections[senderName]
        if (pc) {
            await pc.addIceCandidate(new RTCIceCandidate(candidate))
        }
    }

    function toggleMicrophone() {
        console.log("microphone button")
        if (!localStream) return;
        const audioTrack = localStream.getAudioTracks()[0];
        console.log("AUDIOTRACK:", localStream.getAudioTracks())
        if (audioTrack) {
            console.log("AUDIO TRACK EXISTS")
            audioTrack.enabled = !audioTrack.enabled;
            microphoneButton.className = `control-button mic-button ${audioTrack.enabled ? 'unmuted' : 'muted'}`;
            microphoneButton.textContent = audioTrack.enabled ? "🎙️" : "🎤";

            const playerContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`);
            playerContainer.classList.toggle('speaking', audioTrack.enabled);
        }
    }

    function toggleCamera() {
        if (!localStream) return

        const videoTrack = localStream.getVideoTracks()[0];
        if (videoTrack) {
            console.log("Video tracks")
            videoTrack.enabled = !videoTrack.enabled

            const placeholder = document.querySelector(`.video-container[data-player-name="${currentUserName}"] .video-placeholder`)
            const videoElement = document.querySelector(`.video-container[data-player-name="${currentUserName}"] .video-element`)

            if (videoTrack.enabled) {
                videoElement.style.display = "block"
                placeholder.style.display = "none"
                cameraButton.className = "control-button camera-button on"
            } else {
                videoElement.style.display = "none"
                placeholder.style.display = "flex"
                cameraButton.className = "control-button camera-button off"
            }
        }
    }

    function displayChatMessage(message) {
        const messageDisplay = document.getElementById("messages");
        const messageElement = document.createElement("div");
        messageElement.className = "message";
        messageElement.innerHTML = `
            <div class="message-sender">${message.sender}</div>
            <div class="message-content">${message.content}</div>
            <div class="message-time">${message.timestamp}</div>
        `;
        messageDisplay.appendChild(messageElement);
        messageDisplay.scrollTop = messageDisplay.scrollHeight;
    }

    function checkPlayerCount() {
        const playerCount = playerList.querySelectorAll("li").length;
        const countElement = document.querySelector('.player-count');
        if (countElement) countElement.textContent = playerCount;
        startButton.disabled = playerCount < 4;
        startButton.innerHTML = playerCount >= 4 ? '🎮 Start Game' : `🎮 Need ${4 - playerCount} more players`;
    }

    function getCurrentUserNameFromURL() {
        return new URLSearchParams(window.location.search).get("user")
    }
})