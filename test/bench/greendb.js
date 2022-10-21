import http from "k6/http";

export default function() {
    
    const url = "http://localhost:8081/databases/pgrest/projects?select=id,name,clients(*)";

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Accept-Profile': 'test',
            },
        };
    
    http.get(url, params);
};

