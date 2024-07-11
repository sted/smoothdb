<script lang="ts">
    import RiDatabase2Line from "/assets/images/database-2-line.svg";
    import RiGroupLine from "/assets/images/group-line.svg";
</script>

<aside>
    <div class="logo"><a href="/"><b>smoothdb</b></a></div>
    <ol>
        <li>
            <a href="/ui/databases"><img class="remixicon" src={RiDatabase2Line} /> Databases</a>
        </li>
        <li>
            <a href="/ui/roles"><img class="remixicon" src={RiGroupLine} /> Roles</a>
        </li>
    </ol>
</aside>

<style>
    aside {
        display: flex;
        flex-direction: column;
        overflow-y: auto;
        margin: 0 20px;
    }

    .logo {
        height: 70px;
        margin-top: 25px;
    }
    ol {
        margin-top: 20px;
        list-style: none;
    }

    li {
        padding: 10px 0;
        cursor: pointer;
    }

    li:hover {
        text-decoration: underline;
    }
</style>
