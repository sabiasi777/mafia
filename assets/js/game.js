window.addEventListener("DOMContentLoaded", async () => {
    const playerList = document.getElementById("playerList");
    const roleContainer = document.getElementById('roleContainer');
    const roleText = document.getElementById('roleText');
    const gameArea = document.getElementById('gameArea');
    const sendButton = document.getElementById("sendButton");
    const microphoneButton = document.getElementById("micButton");
    const cameraButton = document.getElementById("cameraButton");
    const finishSpeechButton = document.getElementById("finishSpeechButton")
    const gameData = document.getElementById("gameData");
    const startButton = document.getElementById("startButton");
    const gamePhase = document.querySelector(".game-phase")
    const roomOwner = gameData.dataset.owner;

    const peerConnections = {};

    const configuration = {
        iceServers: [
            { urls: 'stun:stun.l.google.com:19302' },
            { urls: 'stun:stun1.l.google.com:19302' }
        ]
    };

    gamePhase.textContent = "Day Phase"

    cameraButton.disabled = true;
    microphoneButton.disabled = true;

    var roomCode = window.location.pathname.split("/").pop();

    var ws;
    var localStream;
    var currentUserName = getCurrentUserNameFromURL();

    startButton.addEventListener('click', onStartButtonClick);
    sendButton.addEventListener('click', onSendButtonClick);
    microphoneButton.addEventListener('click', toggleMicrophone);
    cameraButton.addEventListener('click', toggleCamera);
    document.getElementById("messageInput").addEventListener("keypress", (e) => {
        if (e.key === "Enter") onSendButtonClick();
    });
    finishSpeechButton.addEventListener("click", () => {
        ws.send(JSON.stringify({ type: "finish-speech" }))
    })

    const observer = new MutationObserver(checkPlayerCount);
    observer.observe(playerList, { childList: true });

    connectToSignalingServer();

    async function onStartButtonClick() {
        if (!localStream) {
            const mediaReady = await setupLocalMedia();
            if (!mediaReady) return;
            
            cameraButton.disabled = false;
            microphoneButton.disabled = false;
            
            await new Promise(resolve => setTimeout(resolve, 500));

            toggleCamera();
            toggleMicrophone();
        }

        ws.send(JSON.stringify({ type: "ready-to-connect" }));

        const isReadyToStart = !startButton.disabled;
        if (currentUserName === roomOwner && isReadyToStart) {
            fetch("/start", {
                method: "POST",
                headers: { 'Content-Type': "application/json" },
                body: JSON.stringify({ roomCode, currentUserName })
            })
                .catch(err => console.error("Failed to start game:", err));
        }

        if (currentUserName !== roomOwner) {
            startButton.textContent = "Waiting for owner...";
            startButton.disabled = true;
        }

    }

    function onSendButtonClick() {
        const input = document.getElementById("messageInput");

        if (input.value.trim() && ws) {
            const messageData = {
                sender: currentUserName,
                content: input.value.trim(),
                timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
                type: "text"
            };

            ws.send(JSON.stringify(messageData));
            input.value = "";
        }
    }

    function startGameUI(me, players) {
        document.getElementById("gameInfo").style.display = "none";
        document.getElementById("statusBar").style.display = "none";
        document.getElementsByTagName("header")[0].style.display = "none";
        startButton.style.display = "none";

        const videoGrid = document.querySelector(".video-grid");
        videoGrid.innerHTML = "";

        players.forEach(player => {
            const videoContainer = document.createElement("div");
            videoContainer.className = "video-container";
            videoContainer.setAttribute("data-player-name", player.name);

            videoContainer.innerHTML = `
                <video class="video-element" playsinline autoplay></video>
                <div class="video-placeholder">
                    <div class="avatar">${player.name.charAt(0).toUpperCase()}</div>
                    <div>${player.name}</div>
                </div>
                <div class="video-overlay">
                    <span class="player-name">${player.name}</span>
                    <div class="video-status">
                        <div class="status-icon mic-off">🎤</div>
                        <div class="status-icon camera-off">📷</div>
                    </div>
                </div>`;
            videoGrid.appendChild(videoContainer);

            if (player.name === me.name) {
                const videoElement = videoContainer.querySelector("video.video-element")
                const placeholder = videoContainer.querySelector(".video-placeholder")

                if (localStream) {
                    videoElement.srcObject = localStream
                    videoElement.muted = true
                    placeholder.style.display = "none"
                    videoElement.style.display = "block"
                }
            } else {
                if (me.name < player.name) {
                    createAndSendOffer(player.name)
                }
            }
        });

        roleText.textContent = `Your role: ${me.role}`;
        roleContainer.style.display = "flex";

        setTimeout(async () => {
            roleContainer.style.display = "none";
            gameArea.style.display = "grid";
        }, 3000);
    }

    async function setupLocalMedia() {
        try {
            localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
            const myVideoContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`);

            if (myVideoContainer) {
                const video = myVideoContainer.querySelector("video.video-element");
                const placeholder = myVideoContainer.querySelector(".video-placeholder");

                placeholder.style.display = 'none';
                video.srcObject = localStream;
                video.muted = true;
                video.style.display = "block";
            }

            return true;
        } catch (err) {
            console.error("Error accessing media devices:", err);
            alert("Could not access your camera and microphone. Please check permissions");
            return false;
        }
    }

    function connectToSignalingServer() {
        const protocol = window.location.protocol === "https:" ? "wss" : "ws";
        const host = window.location.host;
        ws = new WebSocket(`${protocol}://${host}/ws/chat?room=${roomCode}&user=${currentUserName}`);

        ws.onopen = function () {
            console.log("Connected to WebSocket server");
        };

        ws.onmessage = async function (event) {
            console.log("onmessage");
            const message = JSON.parse(event.data);

            switch (message.type) {
                case "text":
                    console.log("Received text");
                    displayChatMessage(message);
                    break;
                case "offer":
                    console.log(`Received offer from ${message.sender}.`);
                    await handleOffer(message.sender, message.sdp);
                    break;
                case "answer":
                    console.log(`Received answer from ${message.sender}.`);
                    await handleAnswer(message.sender, message.sdp);
                    break;
                case "candidate":
                    console.log("Received candidate");
                    await handleIceCandidate(message.sender, message.candidate);
                    break;
                case "player-list-update":
                    console.log("player-list-update");
                    updatePlayerListUI(message);
                    break;
                case "ready-to-connect":
                    if (message.sender !== currentUserName) {
                        console.log(`${message.sender} is ready for video. Sending offer.`);
                        createAndSendOffer(message.sender);
                    }
                    break;
                case "game-start":
                    console.log("game-start");
                    startGameUI(message.me, message.players);
                    break;
            case "turn-update":
                turnUpdate(message.speakerName)
                break;
            case "phase-change":
                if (message.phase === "Night") {
                    // DO SOMETHING
                    showNightUI()
                    gamePhase.textContent = "Night Phase"
                } else if (message.phase === "Day"){
                    // DO SOMETHING
                    gamePhase.textContent = "Day Phase"
                }
                break;
            }
        };

        ws.onclose = function () {
            console.log("WebSocket connection closed, retrying...");
            setTimeout(connectToSignalingServer, 1000);
        };

        ws.onerror = function (error) {
            console.error("WebSocket error:", error);
        };
    }

    function showNightUI() {
        if (myRole === "Mafia" || myRole === "Doctor" || myRole === "Detective") {
            const targetList = document.getElementById('targetList'); // create targetList element
            targetList.innerHTML = players
                .filter(p => p.name !== currentUserName && p.isActive)
                .map(p => `<button class="target-button" data-target="${p.name}">${p.name}</button>`)
                .join('');

            document.querySelectorAll('.target-button').forEach(button => {
                button.addEventListener('click', onTargetSelect);
            });
        }
    }

    function onTargetSelect(event) {
        const targetName = event.target.dataset.target;

        ws.send(JSON.stringify({
            type: 'night-action',
            target: targetName
        }));

        document.getElementById('targetList').innerHTML = `<p>You have made your choice. Waiting...</p>`;
    }

    function turnUpdate(speakerName) {
        console.log(`It's ${speakerName}'s turn to speak.`);
        startRealTimeTimer()

        const allVideoContainers = document.querySelectorAll('.video-container');

        allVideoContainers.forEach(container => {
            container.classList.remove('is-speaking');
        });

        const speakerContainer = document.querySelector(`.video-container[data-player-name="${speakerName}"]`);
        if (speakerContainer) {
            speakerContainer.classList.add('is-speaking');
        }

        // speaker name might not be equal to currentUserName
        console.log("speakerName:", speakerName)
        console.log("currentUserName:", currentUserName)

        if (speakerName === currentUserName) {
            finishSpeechButton.style.display = "block";
            finishSpeechButton.disabled = false;
        } else {
            finishSpeechButton.style.display = "none";
            finishSpeechButton.disabled = true;
        }

        
    }

    function updatePlayerListUI(message) {
        const playerList = document.getElementById("playerList");
        const playerCountElement = document.querySelector(".player-count");
        const roleList = document.getElementById("roleList")

        playerList.innerHTML = "";

        console.log("PLAYERS IN UPDATEPLAYERLISTUI:", message.players);
        console.log("roleList:", roleList)
        console.log("activeRoles:", message.activeRoles)

        if (roleList && message.activeRoles) {
            console.log("Changing active roles list")
            roleList.innerHTML = message.activeRoles.map(role =>
                `<li class="role-item">${role}</li>`).join('');    
        }

        message.players.forEach(player => {
            console.log("Changing player list")
            const li = document.createElement("li");
            li.className = "player-item active";
            li.textContent = player.name;
            playerList.appendChild(li);
        });

        if (localStream) {
            const myVideoContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`);
            if (myVideoContainer) {
                const videoElement = myVideoContainer.querySelector("video.video-element");
                const placeholder = myVideoContainer.querySelector(".video-placeholder");

                videoElement.srcObject = localStream;
                videoElement.muted = true;

                const videoTrack = localStream.getVideoTracks()[0];
                if (videoTrack && videoTrack.enabled) {
                    placeholder.style.display = "none";
                    videoElement.style.display = "block";
                } else {
                    placeholder.style.display = "flex";
                    videoElement.style.display = "none";
                }
            }
        }

        playerCountElement.textContent = message.players.length;
        checkPlayerCount(); // pass the known playerCount directly
    }

    function createPeerConnection(playerName) {
        if (peerConnections[playerName]) return peerConnections[playerName];

        const pc = new RTCPeerConnection(configuration);
        peerConnections[playerName] = pc;

        if (localStream) {
            localStream.getTracks().forEach(track => pc.addTrack(track, localStream));   
        }

        pc.ontrack = (event) => {
            const remoteVideoContainer = document.querySelector(`.video-container[data-player-name="${playerName}"]`);
            if (remoteVideoContainer) {
                const video = remoteVideoContainer.querySelector("video.video-element");
                const placeholder = remoteVideoContainer.querySelector(".video-placeholder");
                placeholder.style.display = "none";
                video.srcObject = event.streams[0];
                video.style.display = "block";
            }
        };

        pc.onicecandidate = (event) => {
            if (event.candidate) {
                ws.send(JSON.stringify({
                    type: 'candidate',
                    sender: currentUserName,
                    receiver: playerName,
                    candidate: event.candidate,
                }));
            }
        };
        return pc;
    }

    async function createAndSendOffer(playerName) {
        if (!localStream) return;
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
        if (!localStream) return;
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
        const pc = peerConnections[senderName];
        if (pc) {
            await pc.setRemoteDescription(new RTCSessionDescription(sdp));
        }
    }

    async function handleIceCandidate(senderName, candidate) {
        const pc = peerConnections[senderName];
        if (pc) {
            await pc.addIceCandidate(new RTCIceCandidate(candidate));
        }
    }

    function toggleMicrophone() {
        if (!localStream) return;
        const audioTrack = localStream.getAudioTracks()[0];
        if (audioTrack) {
            audioTrack.enabled = !audioTrack.enabled;
            microphoneButton.className = `control-button mic-button ${audioTrack.enabled ? 'unmuted' : 'muted'}`;
            microphoneButton.textContent = audioTrack.enabled ? "🎙️" : "🎤";

            const playerContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`);
            if (playerContainer) {
                playerContainer.classList.toggle('speaking', audioTrack.enabled);
            }
        }
    }

    function toggleCamera() {
        if (!localStream) return;

        const videoTrack = localStream.getVideoTracks()[0];
        if (videoTrack) {
            videoTrack.enabled = !videoTrack.enabled;

            const placeholder = document.querySelector(`.video-container[data-player-name="${currentUserName}"] .video-placeholder`);
            const videoElement = document.querySelector(`.video-container[data-player-name="${currentUserName}"] .video-element`);

            if (videoTrack.enabled) {
                videoElement.style.display = "block";
                placeholder.style.display = "none";
                cameraButton.className = "control-button camera-button on";
            } else {
                videoElement.style.display = "none";
                placeholder.style.display = "flex";
                cameraButton.className = "control-button camera-button off";
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
        const warning = document.querySelector(".warning");
        if (countElement) countElement.textContent = playerCount;
        if (warning) warning.style.display = playerCount >= 2 ? "none" : "block";

        const canStart = currentUserName === roomOwner && playerCount >= 2;

        startButton.disabled = !canStart;
        startButton.innerHTML = playerCount >= 2 ? 
            (currentUserName === roomOwner ? '🎮 Start Game' : 'Waiting for owner to start') : 
            `🎮 Need ${2 - playerCount} more players`;
    }

    function getCurrentUserNameFromURL() {
        return new URLSearchParams(window.location.search).get("user");
    }
});

var timeInterval

function startRealTimeTimer() {
    const timer = document.querySelector(".timer")
    var remainingSeconds = 60

    if (timeInterval) {
        clearInterval(timeInterval);
    }
    
    timeInterval = setInterval(() => {
        if (remainingSeconds >= 0) {
            timer.textContent = String(remainingSeconds).padStart(2, '0');
            remainingSeconds--
        } else {
            clearInterval(timeInterval);
            timer.textContent = "00";
        }
    }, 1000);
}
